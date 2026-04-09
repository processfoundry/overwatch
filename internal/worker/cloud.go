package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/christianmscott/overwatch/pkg/spec"
)

// CloudJobSource implements spec.JobSource against the platform Postgres DB.
// Workers claim Monitor rows using SELECT … FOR UPDATE SKIP LOCKED.
type CloudJobSource struct {
	db            *pgxpool.Pool
	workerID      string
	leaseDuration time.Duration
	batchSize     int
}

func NewCloudJobSource(db *pgxpool.Pool, workerID string, leaseDuration time.Duration, batchSize int) *CloudJobSource {
	return &CloudJobSource{
		db:            db,
		workerID:      workerID,
		leaseDuration: leaseDuration,
		batchSize:     batchSize,
	}
}

// Poll claims up to batchSize due monitors and returns them as Leases.
func (s *CloudJobSource) Poll(ctx context.Context, _ spec.WorkerInfo) ([]spec.Lease, error) {
	now := time.Now().UTC()
	expiry := now.Add(s.leaseDuration)

	rows, err := s.db.Query(ctx, `
		UPDATE "Monitor"
		SET "leaseOwner" = $1, "leaseExpiresAt" = $2
		WHERE id IN (
			SELECT id FROM "Monitor"
			WHERE enabled = true
			  AND "nextCheckAt" <= $3
			  AND ("leaseExpiresAt" IS NULL OR "leaseExpiresAt" < $3)
			ORDER BY "nextCheckAt"
			LIMIT $4
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, "orgId", name, type, config, "lastCheckIn", "lastCheckInStatus"
	`, s.workerID, expiry, now, s.batchSize)
	if err != nil {
		return nil, fmt.Errorf("poll: %w", err)
	}
	defer rows.Close()

	var leases []spec.Lease
	for rows.Next() {
		var (
			id, orgID, name, monType string
			rawConfig                []byte
			lastCheckIn              *time.Time
			lastCheckInStatus        *string
		)
		if err := rows.Scan(&id, &orgID, &name, &monType, &rawConfig, &lastCheckIn, &lastCheckInStatus); err != nil {
			return nil, fmt.Errorf("poll scan: %w", err)
		}

		check, err := monitorToCheckSpec(name, monType, rawConfig)
		if err != nil {
			slog.Warn("skipping monitor with unparseable config", "id", id, "type", monType, "error", err)
			_, _ = s.db.Exec(ctx, `UPDATE "Monitor" SET "leaseOwner" = NULL, "leaseExpiresAt" = NULL WHERE id = $1`, id)
			continue
		}

		if monType == "SCHEDULED" {
			check.LastCheckInAt = lastCheckIn
			if lastCheckInStatus != nil {
				check.LastCheckInStatus = *lastCheckInStatus
			}
		}

		leases = append(leases, spec.Lease{
			ID:        id + "-" + now.Format("20060102T150405"),
			MonitorID: id,
			OrgID:     orgID,
			Check:     check,
			IssuedAt:  now,
			ExpiresAt: expiry,
			Token:     s.workerID,
		})
	}
	return leases, rows.Err()
}

// Ack commits a completed check: writes the result, updates Monitor status,
// advances nextCheckAt, clears the lease, and dispatches alerts on transitions.
func (s *CloudJobSource) Ack(ctx context.Context, lease spec.Lease, result spec.CheckResult) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ack begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Read current status + interval config so we can detect transitions and compute nextCheckAt.
	var prevStatus string
	var rawConfig []byte
	err = tx.QueryRow(ctx,
		`SELECT status, config FROM "Monitor" WHERE id = $1 AND "leaseOwner" = $2`,
		lease.MonitorID, s.workerID,
	).Scan(&prevStatus, &rawConfig)
	if err == pgx.ErrNoRows {
		// Lease was stolen or monitor deleted; silently discard.
		return tx.Rollback(ctx)
	}
	if err != nil {
		return fmt.Errorf("ack read monitor: %w", err)
	}

	newStatus := string(result.Status)
	statusChanged := prevStatus != newStatus
	var statusChangedAt *time.Time
	if statusChanged {
		t := result.Timestamp.UTC()
		statusChangedAt = &t
	}

	interval := intervalFromConfig(rawConfig, lease.Check.Interval.Duration)
	nextCheckAt := result.Timestamp.UTC().Add(interval)
	latencyMs := int(result.Duration.Milliseconds())

	detailMap := make(map[string]any, len(result.Detail)+1)
	for k, v := range result.Detail {
		detailMap[k] = v
	}
	if result.Error != "" {
		detailMap["error"] = result.Error
	}
	detail, _ := json.Marshal(detailMap)

	var alertTriggered bool
	var alertOutcome any

	if statusChanged {
		alertTriggered, alertOutcome = dispatchAlerts(ctx, tx, lease, result, prevStatus)
	}

	alertOutcomeJSON, _ := json.Marshal(alertOutcome)

	// Insert CheckResult.
	_, err = tx.Exec(ctx, `
		INSERT INTO "CheckResult" (id, "monitorId", status, "latencyMs", detail, "alertTriggered", "alertOutcome", "createdAt")
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7)
	`, lease.MonitorID, newStatus, latencyMs, detail, alertTriggered, alertOutcomeJSON, result.Timestamp.UTC())
	if err != nil {
		return fmt.Errorf("ack insert result: %w", err)
	}

	// Update Monitor.
	_, err = tx.Exec(ctx, `
		UPDATE "Monitor"
		SET status = $1,
		    "statusChangedAt" = COALESCE($2, "statusChangedAt"),
		    "nextCheckAt" = $3,
		    "leaseOwner" = NULL,
		    "leaseExpiresAt" = NULL,
		    "updatedAt" = NOW()
		WHERE id = $4
	`, newStatus, statusChangedAt, nextCheckAt, lease.MonitorID)
	if err != nil {
		return fmt.Errorf("ack update monitor: %w", err)
	}

	return tx.Commit(ctx)
}

// Nack releases the lease and applies a short backoff before retry.
func (s *CloudJobSource) Nack(ctx context.Context, lease spec.Lease, reason string) error {
	slog.Warn("nack", "monitor", lease.MonitorID, "reason", reason)
	_, err := s.db.Exec(ctx, `
		UPDATE "Monitor"
		SET "leaseOwner" = NULL, "leaseExpiresAt" = NULL,
		    "nextCheckAt" = NOW() + interval '30 seconds'
		WHERE id = $1 AND "leaseOwner" = $2
	`, lease.MonitorID, s.workerID)
	return err
}

// Heartbeat extends the lease expiry to prevent expiration during long-running checks.
func (s *CloudJobSource) Heartbeat(ctx context.Context, lease spec.Lease) error {
	expiry := time.Now().UTC().Add(s.leaseDuration)
	_, err := s.db.Exec(ctx, `
		UPDATE "Monitor" SET "leaseExpiresAt" = $1
		WHERE id = $2 AND "leaseOwner" = $3
	`, expiry, lease.MonitorID, s.workerID)
	return err
}

// ── Config mapping ────────────────────────────────────────────────────────────

type monitorConfig map[string]any

func monitorToCheckSpec(name, monType string, rawConfig []byte) (spec.CheckSpec, error) {
	var cfg monitorConfig
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		return spec.CheckSpec{}, fmt.Errorf("unmarshal config: %w", err)
	}

	interval := spec.Duration{Duration: time.Duration(cfgInt(cfg, "interval", 30)) * time.Second}
	timeout := spec.Duration{Duration: time.Duration(cfgInt(cfg, "timeout", 10)) * time.Second}

	check := spec.CheckSpec{
		Name:     name,
		Interval: interval,
		Timeout:  timeout,
	}

	switch monType {
	case "HTTP":
		check.Type = spec.CheckHTTP
		check.Target = cfgStr(cfg, "url", "")
		check.Headers = cfgHeaders(cfg)
		check.ExpectedStatus = cfgInt(cfg, "expectedStatus", 200)
		check.LatencyThresholdMs = cfgInt(cfg, "latencyThresholdMs", 0)

	case "TCP":
		check.Type = spec.CheckTCP
		host := cfgStr(cfg, "host", "")
		port := cfgInt(cfg, "port", 80)
		check.Target = fmt.Sprintf("%s:%d", host, port)
		check.LatencyThresholdMs = cfgInt(cfg, "latencyThresholdMs", 0)

	case "TLS":
		check.Type = spec.CheckTLS
		check.Target = cfgStr(cfg, "hostname", "")
		check.WarnDays = cfgInt(cfg, "warnDays", 7)

	case "DNS":
		check.Type = spec.CheckDNS
		check.Target = cfgStr(cfg, "domain", "")
		check.RecordType = cfgStr(cfg, "recordType", "A")

	case "SCHEDULED":
		check.Type = spec.CheckCheckIn
		expectedInterval := cfgStr(cfg, "expectedInterval", "1h")
		d, err := time.ParseDuration(expectedInterval)
		if err != nil {
			d = time.Hour
		}
		gracePeriod := cfgStr(cfg, "gracePeriod", "0")
		gd, err := time.ParseDuration(gracePeriod)
		if err != nil {
			gd = 0
		}
		check.MaxSilence = spec.Duration{Duration: d + gd}

	default:
		return spec.CheckSpec{}, fmt.Errorf("unknown monitor type %q", monType)
	}

	return check, nil
}

func intervalFromConfig(rawConfig []byte, fallback time.Duration) time.Duration {
	var cfg monitorConfig
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		return fallback
	}
	secs := cfgInt(cfg, "interval", 0)
	if secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return fallback
}

func cfgStr(cfg monitorConfig, key, def string) string {
	if v, ok := cfg[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

func cfgInt(cfg monitorConfig, key string, def int) int {
	if v, ok := cfg[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return def
}

func cfgHeaders(cfg monitorConfig) map[string]string {
	raw, ok := cfg["headers"]
	if !ok {
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
