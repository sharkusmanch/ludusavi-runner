package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sharkusmanch/ludusavi-runner/internal/app"
	"github.com/sharkusmanch/ludusavi-runner/internal/executor"
	"github.com/sharkusmanch/ludusavi-runner/internal/http"
	"github.com/sharkusmanch/ludusavi-runner/internal/metrics"
	"github.com/sharkusmanch/ludusavi-runner/internal/notify"
	"github.com/spf13/cobra"
)

// NewServeCmd creates the serve command.
func NewServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the service in foreground",
		Long: `Run the backup service in foreground mode.

This runs the scheduler loop, executing backups at the configured interval.
Use Ctrl+C to stop.

This is useful for debugging or running in a container.`,
		RunE: runServe,
	}

	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger, err := setupLogging(cfg)
	if err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}
	logger.Info("starting ludusavi-runner in foreground mode")

	// Create HTTP client with retry config
	httpClient := http.NewClient(
		http.WithRetryConfig(http.RetryConfig{
			MaxAttempts:  cfg.Retry.MaxAttempts,
			InitialDelay: cfg.Retry.InitialDelay,
			MaxDelay:     cfg.Retry.MaxDelay,
		}),
		http.WithLogger(logger),
	)

	// Create executor
	execOpts := []executor.LudusaviOption{
		executor.WithLogger(logger),
	}
	if cfg.LudusaviPath != "" {
		execOpts = append(execOpts, executor.WithBinaryPath(cfg.LudusaviPath))
	}
	exec := executor.NewLudusaviExecutor(execOpts...)

	// Create runner
	runnerOpts := []app.RunnerOption{
		app.WithExecutor(exec),
		app.WithLogger(logger),
	}

	// Create metrics pusher if enabled
	if cfg.Metrics.Enabled {
		metricsPusher := metrics.NewPushgatewayClient(
			cfg.Metrics.PushgatewayURL,
			metrics.WithHTTPClient(httpClient),
			metrics.WithLogger(logger),
		)
		runnerOpts = append(runnerOpts, app.WithMetricsPusher(metricsPusher))
	}

	// Create notifier if enabled
	if cfg.Apprise.Enabled {
		notifier := notify.NewAppriseClient(
			cfg.Apprise.URL,
			cfg.Apprise.Key,
			notify.WithHTTPClient(httpClient),
			notify.WithLogger(logger),
		)
		runnerOpts = append(runnerOpts, app.WithNotifier(notifier))
	}

	runner := app.NewRunner(cfg, runnerOpts...)

	// Create scheduler
	scheduler := app.NewScheduler(runner,
		app.WithInterval(cfg.Interval),
		app.WithBackupOnStartup(cfg.BackupOnStartup),
		app.WithSchedulerLogger(logger),
	)

	// Set up signal handling
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	// Start scheduler
	if err := scheduler.Start(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("scheduler error: %w", err)
	}

	logger.Info("ludusavi-runner stopped")
	return nil
}
