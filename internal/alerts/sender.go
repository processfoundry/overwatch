package alerts

import (
	"context"

	"github.com/christianmscott/overwatch/pkg/spec"
)

type AlertSender interface {
	Send(ctx context.Context, msg spec.AlertMessage) error
	Name() string
}
