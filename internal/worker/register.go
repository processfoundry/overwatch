package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/christianmscott/overwatch/internal/version"
)

const heartbeatInterval = 15 * time.Second

// Register upserts the worker row and then loops sending heartbeats until ctx is cancelled.
// On clean shutdown it marks the worker as stopped.
func Register(ctx context.Context, db *pgxpool.Pool, workerID, region string, concurrency int) {
	_, err := db.Exec(ctx, `
		INSERT INTO "Worker" ("workerId", version, region, status, concurrency, "lastHeartbeat", "startedAt")
		VALUES ($1, $2, $3, 'active', $4, NOW(), NOW())
		ON CONFLICT ("workerId") DO UPDATE
		SET version = EXCLUDED.version,
		    region = EXCLUDED.region,
		    status = 'active',
		    concurrency = EXCLUDED.concurrency,
		    "lastHeartbeat" = NOW(),
		    "startedAt" = NOW()
	`, workerID, version.Version, region, concurrency)
	if err != nil {
		slog.Error("worker: register failed", "error", err)
		return
	}
	slog.Info("worker registered", "id", workerID, "version", version.Version, "region", region)

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, err := db.Exec(stopCtx, `UPDATE "Worker" SET status = 'stopped' WHERE "workerId" = $1`, workerID)
			if err != nil {
				slog.Error("worker: stop update failed", "error", err)
			}
			return
		case <-ticker.C:
			_, err := db.Exec(ctx, `UPDATE "Worker" SET "lastHeartbeat" = NOW() WHERE "workerId" = $1`, workerID)
			if err != nil {
				slog.Error("worker: heartbeat failed", "error", err)
			}
		}
	}
}
