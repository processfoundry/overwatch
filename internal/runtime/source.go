package runtime

import (
	"context"

	"github.com/christianmscott/overwatch/pkg/spec"
)

type JobSource interface {
	Poll(ctx context.Context, worker spec.WorkerInfo) ([]spec.Lease, error)
	Ack(ctx context.Context, lease spec.Lease, result spec.CheckResult) error
	Nack(ctx context.Context, lease spec.Lease, reason string) error
	Heartbeat(ctx context.Context, lease spec.Lease) error
}
