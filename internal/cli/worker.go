package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"github.com/christianmscott/overwatch/internal/scheduler"
	"github.com/christianmscott/overwatch/internal/version"
	"github.com/christianmscott/overwatch/internal/worker"
	"github.com/christianmscott/overwatch/pkg/spec"
)

var workerCmd = &cobra.Command{
	Use:    "worker",
	Short:  "Run as a managed worker (Overwatch Cloud)",
	Hidden: true,
	RunE:   runWorker,
}

var (
	workerDBURL         string
	workerConcurrency   int
	workerPollInterval  time.Duration
	workerLeaseDuration time.Duration
	workerBatchSize     int
	workerRegion        string
)

func init() {
	workerCmd.Flags().StringVar(&workerDBURL, "db-url", "", "Postgres connection string (or DATABASE_URL env)")
	workerCmd.Flags().IntVar(&workerConcurrency, "concurrency", 10, "Max concurrent checks")
	workerCmd.Flags().DurationVar(&workerPollInterval, "poll-interval", 5*time.Second, "Poll interval")
	workerCmd.Flags().DurationVar(&workerLeaseDuration, "lease-duration", 60*time.Second, "Lease TTL")
	workerCmd.Flags().IntVar(&workerBatchSize, "batch-size", 20, "Max monitors to claim per poll")
	workerCmd.Flags().StringVar(&workerRegion, "region", "", "Worker region label (optional)")
}

func runWorker(cmd *cobra.Command, _ []string) error {
	dbURL := workerDBURL
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		return fmt.Errorf("--db-url or DATABASE_URL is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	workerID := workerInstanceID()
	region := workerRegion
	if region == "" {
		region = os.Getenv("WORKER_REGION")
	}
	if region == "" {
		region, _ = os.Hostname()
	}

	slog.Info("starting worker",
		"id", workerID,
		"version", version.Version,
		"region", region,
		"concurrency", workerConcurrency,
		"poll_interval", workerPollInterval,
		"lease_duration", workerLeaseDuration,
		"batch_size", workerBatchSize,
	)

	source := worker.NewCloudJobSource(db, workerID, workerLeaseDuration, workerBatchSize)

	wi := spec.WorkerInfo{
		ID:      workerID,
		Version: version.Version,
	}

	sched := scheduler.New(source, wi, workerPollInterval, workerBatchSize*2)
	pool := worker.NewPool(workerConcurrency, source, nil, workerLeaseDuration)

	go worker.Register(ctx, db, workerID, region, workerConcurrency)
	go sched.Run(ctx)
	pool.Run(ctx, sched.C())

	slog.Info("worker stopped", "id", workerID)
	return nil
}

func workerInstanceID() string {
	if v := os.Getenv("WORKER_ID"); v != "" {
		return v
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return fmt.Sprintf("worker-%d", time.Now().UnixMilli())
}
