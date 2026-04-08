package checks

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/christianmscott/overwatch/pkg/spec"
)

type TCPChecker struct{}

func (t *TCPChecker) Check(ctx context.Context, check spec.CheckSpec) spec.CheckResult {
	start := time.Now()
	result := spec.CheckResult{
		CheckName: check.Name,
		Timestamp: start,
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", check.Target)
	result.Duration = time.Since(start)
	if err != nil {
		result.Status = spec.StatusDown
		result.Error = err.Error()
		return result
	}
	conn.Close()

	result.Status = spec.StatusUp
	result.Detail = map[string]any{"target": check.Target}

	if check.LatencyThresholdMs > 0 && result.Duration.Milliseconds() > int64(check.LatencyThresholdMs) {
		result.Status = spec.StatusDegraded
		result.Error = fmt.Sprintf("response time %dms exceeds threshold %dms",
			result.Duration.Milliseconds(), check.LatencyThresholdMs)
	}

	return result
}
