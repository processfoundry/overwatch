package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/christianmscott/overwatch/pkg/spec"
)

type Scheduler struct {
	source   spec.JobSource
	worker   spec.WorkerInfo
	interval time.Duration
	out      chan spec.Lease
}

func New(source spec.JobSource, worker spec.WorkerInfo, tickInterval time.Duration, buffer int) *Scheduler {
	return &Scheduler{
		source:   source,
		worker:   worker,
		interval: tickInterval,
		out:      make(chan spec.Lease, buffer),
	}
}

func (s *Scheduler) C() <-chan spec.Lease {
	return s.out
}

func (s *Scheduler) Run(ctx context.Context) {
	s.poll(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	defer close(s.out)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.poll(ctx)
		}
	}
}

func (s *Scheduler) poll(ctx context.Context) {
	leases, err := s.source.Poll(ctx, s.worker)
	if err != nil {
		slog.Error("poll failed", "error", err)
		return
	}
	for _, l := range leases {
		select {
		case s.out <- l:
		case <-ctx.Done():
			return
		}
	}
}
