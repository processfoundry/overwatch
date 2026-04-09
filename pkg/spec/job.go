package spec

import (
	"context"
	"time"
)

type Lease struct {
	ID        string    `json:"id"`
	MonitorID string    `json:"monitor_id"`
	OrgID     string    `json:"org_id"`
	Check     CheckSpec `json:"check"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Token     string    `json:"token"`
}

type WorkerInfo struct {
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

type JobSource interface {
	Poll(ctx context.Context, worker WorkerInfo) ([]Lease, error)
	Ack(ctx context.Context, lease Lease, result CheckResult) error
	Nack(ctx context.Context, lease Lease, reason string) error
	Heartbeat(ctx context.Context, lease Lease) error
}
