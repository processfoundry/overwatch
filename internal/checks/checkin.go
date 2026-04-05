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
		return result
	}

	result.Status = spec.StatusUp
	return result
}

var DefaultCheckIn = &CheckInChecker{}
