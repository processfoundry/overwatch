package runtime

import (
	"context"
	"sync"
	"time"

	"github.com/christianmscott/overwatch/pkg/spec"
)

type LocalJobSource struct {
	checks []spec.CheckSpec

	mu      sync.Mutex
	nextRun map[string]time.Time
}

func NewLocalJobSource(checks []spec.CheckSpec) *LocalJobSource {
	next := make(map[string]time.Time, len(checks))
	now := time.Now()
	for _, c := range checks {
		next[c.Name] = now
	}
	return &LocalJobSource{checks: checks, nextRun: next}
}

func (s *LocalJobSource) Poll(_ context.Context, _ spec.WorkerInfo) ([]spec.Lease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var leases []spec.Lease

	for _, c := range s.checks {
		if !now.Before(s.nextRun[c.Name]) {
			s.nextRun[c.Name] = now.Add(c.Interval.Duration)
			leases = append(leases, spec.Lease{
				ID:        c.Name + "-" + now.Format("20060102T150405"),
				Check:     c,
				IssuedAt:  now,
				ExpiresAt: now.Add(c.Timeout.Duration * 2),
				Token:     "local",
			})
		}
	}

	return leases, nil
}

func (s *LocalJobSource) UpdateChecks(checks []spec.CheckSpec) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing := make(map[string]time.Time, len(s.nextRun))
	for k, v := range s.nextRun {
		existing[k] = v
	}

	s.checks = checks
	s.nextRun = make(map[string]time.Time, len(checks))
	now := time.Now()
	for _, c := range checks {
		if t, ok := existing[c.Name]; ok {
			s.nextRun[c.Name] = t
		} else {
			s.nextRun[c.Name] = now
		}
	}
}

func (s *LocalJobSource) Ack(_ context.Context, _ spec.Lease, _ spec.CheckResult) error {
	return nil
}

func (s *LocalJobSource) Nack(_ context.Context, _ spec.Lease, _ string) error {
	return nil
}

func (s *LocalJobSource) Heartbeat(_ context.Context, _ spec.Lease) error {
	return nil
}
