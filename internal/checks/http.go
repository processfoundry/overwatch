package checks

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/christianmscott/overwatch/pkg/spec"
)

type HTTPChecker struct{}

func (h *HTTPChecker) Check(ctx context.Context, check spec.CheckSpec) spec.CheckResult {
	start := time.Now()
	result := spec.CheckResult{
		CheckName: check.Name,
		Timestamp: start,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, check.Target, nil)
	if err != nil {
		result.Status = spec.StatusDown
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	for k, v := range check.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	result.Duration = time.Since(start)
	if err != nil {
		result.Status = spec.StatusDown
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	expected := check.ExpectedStatus
	if expected == 0 {
		expected = 200
	}

	result.Detail = map[string]any{
		"statusCode":    resp.StatusCode,
		"statusText":    resp.Status,
		"contentType":   resp.Header.Get("Content-Type"),
		"contentLength": resp.ContentLength,
		"server":        resp.Header.Get("Server"),
	}

	if resp.StatusCode == expected {
		result.Status = spec.StatusUp
	} else {
		result.Status = spec.StatusDown
		result.Error = "unexpected status " + resp.Status
	}

	if result.Status == spec.StatusUp && check.LatencyThresholdMs > 0 &&
		result.Duration.Milliseconds() > int64(check.LatencyThresholdMs) {
		result.Status = spec.StatusDegraded
		result.Error = fmt.Sprintf("response time %dms exceeds threshold %dms",
			result.Duration.Milliseconds(), check.LatencyThresholdMs)
	}

	return result
}
