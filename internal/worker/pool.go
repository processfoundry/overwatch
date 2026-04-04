package worker

import (
	"context"
	"log/slog"
	"sync"

	"github.com/christianmscott/overwatch/internal/checks"
	"github.com/christianmscott/overwatch/pkg/spec"
)

type ResultHandler func(spec.CheckResult)

type Pool struct {
	concurrency int
	source      spec.JobSource
	onResult    ResultHandler
}

func NewPool(concurrency int, source spec.JobSource, onResult ResultHandler) *Pool {
	return &Pool{
		concurrency: concurrency,
		source:      source,
		onResult:    onResult,
	}
}

func (p *Pool) Run(ctx context.Context, leases <-chan spec.Lease) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, p.concurrency)

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

			sem <- struct{}{}
			wg.Add(1)
			go func(l spec.Lease) {
				defer wg.Done()
				defer func() { <-sem }()

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
