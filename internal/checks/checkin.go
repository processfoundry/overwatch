package checks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/christianmscott/overwatch/pkg/spec"
)

type CheckInChecker struct {
	mu       sync.Mutex
	lastPing map[string]time.Time
}

func (c *CheckInChecker) RecordPing(checkName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lastPing == nil {
		c.lastPing = make(map[string]time.Time)
	}
	c.lastPing[checkName] = time.Now()
}

func (c *CheckInChecker) RecordFailure(checkName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lastPing == nil {
		c.lastPing = make(map[string]time.Time)
	}
	delete(c.lastPing, checkName)
}

func (c *CheckInChecker) Check(_ context.Context, check spec.CheckSpec) spec.CheckResult {
	start := time.Now()
	result := spec.CheckResult{
		CheckName: check.Name,
		Timestamp: start,
		Duration:  0,
	}

	// Cloud mode: lastCheckIn is populated from DB by the job source.
	if check.LastCheckInAt != nil {
		if check.LastCheckInStatus == "fail" {
			result.Status = spec.StatusDown
			result.Error = "last check-in reported failure"
			result.Detail = map[string]any{
				"lastCheckIn": check.LastCheckInAt.Format(time.RFC3339),
				"status":      "fail",
			}
			return result
		}
		silence := time.Since(*check.LastCheckInAt)
		if silence > check.MaxSilence.Duration {
			result.Status = spec.StatusDown
			result.Error = fmt.Sprintf("last check-in %s ago (max %s)", silence.Round(time.Second), check.MaxSilence.Duration)
			result.Detail = map[string]any{
				"lastCheckIn": check.LastCheckInAt.Format(time.RFC3339),
				"silenceFor":  silence.Round(time.Second).String(),
				"maxSilence":  check.MaxSilence.Duration.String(),
			}
			return result
		}
		result.Status = spec.StatusUp
		result.Detail = map[string]any{
			"lastCheckIn": check.LastCheckInAt.Format(time.RFC3339),
			"silenceFor":  silence.Round(time.Second).String(),
			"maxSilence":  check.MaxSilence.Duration.String(),
		}
		return result
	}

	// Self-hosted mode: use in-memory ping map.
	c.mu.Lock()
	last, ok := c.lastPing[check.Name]
	c.mu.Unlock()

	if !ok {
		result.Status = spec.StatusDown
		result.Error = "no check-in received"
		return result
	}

	silence := time.Since(last)
	if silence > check.MaxSilence.Duration {
		result.Status = spec.StatusDown
		result.Error = fmt.Sprintf("last check-in %s ago (max %s)", silence.Round(time.Second), check.MaxSilence.Duration)
		result.Detail = map[string]any{
			"lastCheckIn": last.Format(time.RFC3339),
			"silenceFor":  silence.Round(time.Second).String(),
			"maxSilence":  check.MaxSilence.Duration.String(),
		}
		return result
	}

	result.Status = spec.StatusUp
	result.Detail = map[string]any{
		"lastCheckIn": last.Format(time.RFC3339),
		"silenceFor":  silence.Round(time.Second).String(),
		"maxSilence":  check.MaxSilence.Duration.String(),
	}
	return result
}

var DefaultCheckIn = &CheckInChecker{}
