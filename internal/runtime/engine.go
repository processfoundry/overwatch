package runtime

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/christianmscott/overwatch/internal/alerts"
	"github.com/christianmscott/overwatch/internal/api"
	"github.com/christianmscott/overwatch/internal/config"
	"github.com/christianmscott/overwatch/internal/results"
	"github.com/christianmscott/overwatch/internal/scheduler"
	"github.com/christianmscott/overwatch/internal/version"
	"github.com/christianmscott/overwatch/internal/worker"
	"github.com/christianmscott/overwatch/pkg/spec"
	"github.com/fsnotify/fsnotify"
)

type Engine struct {
	cfg     *spec.Config
	cfgPath string
}

func NewEngine(cfg *spec.Config, cfgPath string) *Engine {
	return &Engine{cfg: cfg, cfgPath: cfgPath}
}

func (e *Engine) Run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := results.NewStore(100)
	source := NewLocalJobSource(e.cfg.Checks)
	router := alerts.NewRouter(alerts.BuildSenders(e.cfg.Alerts))
	srv := api.New(e.cfg, e.cfgPath, store)

	apiReload := make(chan struct{}, 1)
	srv.OnReload(func() {
		select {
		case apiReload <- struct{}{}:
		default:
		}
	})

	stopWatcher := e.startReloadWatcher(ctx, srv, source, router, apiReload)

	go func() {
		if err := srv.Serve(ctx); err != nil {
			slog.Error("api server error", "error", err)
		}
	}()

	wi := spec.WorkerInfo{
		ID:      hostname(),
		Version: version.Version,
	}

	tick := 1 * time.Second
	sched := scheduler.New(source, wi, tick, len(e.cfg.Checks)*2+8)

	if senders := alerts.BuildSenders(e.cfg.Alerts); len(senders) > 0 {
		slog.Info("alerting enabled", "senders", len(senders))
	}

	handleResult := func(r spec.CheckResult) {
		store.Record(r)

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

		router.Handle(r)
	}

	pool := worker.NewPool(e.cfg.Server.Concurrency, source, handleResult, 60*time.Second)

	slog.Info("starting overwatch",
		"checks", len(e.cfg.Checks),
		"concurrency", e.cfg.Server.Concurrency,
		"api", srv.Addr(),
		"version", version.Version,
	)

	go sched.Run(ctx)
	pool.Run(ctx, sched.C())

	close(stopWatcher)
	slog.Info("shutting down")
	return nil
}

// startReloadWatcher listens for SIGHUP and filesystem changes on the config
// file, then propagates the new config to all runtime components. Returns a
// channel that should be closed on shutdown to stop the watcher.
func (e *Engine) startReloadWatcher(ctx context.Context, srv *api.Server, source *LocalJobSource, router *alerts.Router, apiReload <-chan struct{}) chan struct{} {
	done := make(chan struct{})

	doReload := func(reason string) {
		slog.Info("reloading config", "reason", reason, "path", e.cfgPath)
		newCfg, err := config.Load(e.cfgPath)
		if err != nil {
			slog.Error("config reload failed", "error", err)
			return
		}
		e.cfg = newCfg
		srv.UpdateConfig(newCfg)
		source.UpdateChecks(newCfg.Checks)
		router.UpdateSenders(alerts.BuildSenders(newCfg.Alerts))
		slog.Info("config reloaded", "checks", len(newCfg.Checks), "webhooks", len(newCfg.Alerts.Webhooks))
	}

	// Tracks the last time an API-triggered reload occurred so we can
	// suppress the redundant file-watcher event that follows the YAML write.
	var lastAPIReload atomic.Int64

	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Warn("could not start file watcher, only SIGHUP and API reload available", "error", err)
		go func() {
			for {
				select {
				case <-done:
					return
				case <-ctx.Done():
					return
				case <-sighup:
					doReload("SIGHUP")
				case <-apiReload:
					lastAPIReload.Store(time.Now().UnixNano())
					doReload("API")
				}
			}
		}()
		return done
	}

	go func() {
		defer watcher.Close()
		var debounce <-chan time.Time
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-sighup:
				doReload("SIGHUP")
			case <-apiReload:
				lastAPIReload.Store(time.Now().UnixNano())
				doReload("API")
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					debounce = time.After(500 * time.Millisecond)
				}
			case <-debounce:
				if time.Since(time.Unix(0, lastAPIReload.Load())) < 2*time.Second {
					slog.Debug("skipping file-triggered reload (recent API write)")
					continue
				}
				doReload("file change")
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				slog.Warn("file watcher error", "error", err)
			}
		}
	}()

	if err := watcher.Add(e.cfgPath); err != nil {
		slog.Warn("could not watch config file", "path", e.cfgPath, "error", err)
	} else {
		slog.Info("watching config file for changes", "path", e.cfgPath)
	}

	return done
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
