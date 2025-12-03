package cli

import (
	"fmt"
	"log/slog"

	"github.com/sharkusmanch/ludusavi-runner/internal/app"
	"github.com/sharkusmanch/ludusavi-runner/internal/executor"
	"github.com/sharkusmanch/ludusavi-runner/internal/http"
	"github.com/sharkusmanch/ludusavi-runner/internal/metrics"
	"github.com/sharkusmanch/ludusavi-runner/internal/notify"
	"github.com/spf13/cobra"
)

// NewRunCmd creates the run command.
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a single backup cycle and exit",
		Long: `Run a single backup cycle (cloud upload + local backup) and exit.

This is useful for testing or one-off backups.`,
		RunE: runRun,
	}

	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger := slog.Default()

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

	// Create metrics pusher
	metricsPusher := metrics.NewPushgatewayClient(
		cfg.PushgatewayURL,
		metrics.WithHTTPClient(httpClient),
		metrics.WithLogger(logger),
	)

	// Create runner
	runnerOpts := []app.RunnerOption{
		app.WithExecutor(exec),
		app.WithMetricsPusher(metricsPusher),
		app.WithLogger(logger),
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

	// Run backup
	result, err := runner.Run(cmd.Context())
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("backup completed with errors")
	}

	logger.Info("backup completed successfully",
		"duration", result.Duration,
	)

	return nil
}
