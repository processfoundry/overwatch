package runtime

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/christianmscott/overwatch/internal/scheduler"
	"github.com/christianmscott/overwatch/internal/version"
	"github.com/christianmscott/overwatch/internal/worker"
	"github.com/christianmscott/overwatch/pkg/spec"
)

type Engine struct {
	cfg *spec.Config
}

func NewEngine(cfg *spec.Config) *Engine {
	return &Engine{cfg: cfg}
}

func (e *Engine) Run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	source := NewLocalJobSource(e.cfg.Checks)

	wi := spec.WorkerInfo{
		ID:      hostname(),
		Version: version.Version,
	}

	tick := 1 * time.Second
	sched := scheduler.New(source, wi, tick, len(e.cfg.Checks)*2)

	logResult := func(r spec.CheckResult) {
		attrs := []any{
			"check", r.CheckName,
			"status", r.Status,
			"duration", r.Duration,
		}
		if r.Error != "" {
			attrs = append(attrs, "error", r.Error)
		}
		switch r.Status {
		case spec.StatusUp:
			slog.Info("check complete", attrs...)
		case spec.StatusDegraded:
			slog.Warn("check complete", attrs...)
		default:
			slog.Error("check complete", attrs...)
		}
	}

	pool := worker.NewPool(e.cfg.Worker.Concurrency, source, logResult)

	slog.Info("starting overwatch",
		"checks", len(e.cfg.Checks),
		"concurrency", e.cfg.Worker.Concurrency,
		"version", version.Version,
	)

	go sched.Run(ctx)
	pool.Run(ctx, sched.C())

	slog.Info("shutting down")
	return nil
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
