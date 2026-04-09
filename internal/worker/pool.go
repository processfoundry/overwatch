package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/christianmscott/overwatch/internal/checks"
	"github.com/christianmscott/overwatch/pkg/spec"
)

type ResultHandler func(spec.CheckResult)

type Pool struct {
	concurrency   int
	source        spec.JobSource
	onResult      ResultHandler
	leaseDuration time.Duration
}

func NewPool(concurrency int, source spec.JobSource, onResult ResultHandler, leaseDuration time.Duration) *Pool {
	return &Pool{
		concurrency:   concurrency,
		source:        source,
		onResult:      onResult,
		leaseDuration: leaseDuration,
	}
}

func (p *Pool) Run(ctx context.Context, leases <-chan spec.Lease) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, p.concurrency)

	// Track in-flight leases for heartbeating.
	var mu sync.Mutex
	inFlight := make(map[string]spec.Lease)

	heartbeatInterval := p.leaseDuration / 3
	if heartbeatInterval < time.Second {
		heartbeatInterval = time.Second
	}

	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.Lock()
				snapshot := make([]spec.Lease, 0, len(inFlight))
				for _, l := range inFlight {
					snapshot = append(snapshot, l)
				}
				mu.Unlock()

				for _, l := range snapshot {
					if err := p.source.Heartbeat(ctx, l); err != nil {
						slog.Warn("heartbeat failed", "lease", l.ID, "error", err)
					}
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case lease, ok := <-leases:
			if !ok {
				wg.Wait()
				return
			}

			mu.Lock()
			inFlight[lease.ID] = lease
			mu.Unlock()

			sem <- struct{}{}
			wg.Add(1)
			go func(l spec.Lease) {
				defer wg.Done()
				defer func() { <-sem }()
				defer func() {
					mu.Lock()
					delete(inFlight, l.ID)
					mu.Unlock()
				}()

				result := checks.Run(ctx, l.Check)

				if err := p.source.Ack(ctx, l, result); err != nil {
					slog.Error("ack failed", "check", l.Check.Name, "error", err)
				}

				if p.onResult != nil {
					p.onResult(result)
				}
			}(lease)
		}
	}
}
