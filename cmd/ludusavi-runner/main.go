// Package main is the entry point for ludusavi-runner.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/sharkusmanch/ludusavi-runner/internal/app"
	"github.com/sharkusmanch/ludusavi-runner/internal/cli"
	"github.com/sharkusmanch/ludusavi-runner/internal/config"
	"github.com/sharkusmanch/ludusavi-runner/internal/executor"
	"github.com/sharkusmanch/ludusavi-runner/internal/http"
	"github.com/sharkusmanch/ludusavi-runner/internal/metrics"
	"github.com/sharkusmanch/ludusavi-runner/internal/notify"
	"github.com/sharkusmanch/ludusavi-runner/internal/platform"
)

func main() {
	// Check if running as a Windows service
	if platform.IsRunningAsService() {
		if err := runAsService(); err != nil {
			slog.Error("service failed", "error", err)
			os.Exit(1)
		}
		return
	}

	// Run CLI
	cli.Execute()
}

// runAsService runs the application as a Windows service.
func runAsService() error {
	return platform.RunAsService(func(ctx context.Context) error {
		// Load config
		loader := config.NewLoader()
		cfg, err := loader.Load()
		if err != nil {
			return err
		}

		// Set up logging
		logger := slog.Default()

		// Create HTTP client
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

		// Create and start scheduler
		scheduler := app.NewScheduler(runner,
			app.WithInterval(cfg.Interval),
			app.WithBackupOnStartup(cfg.BackupOnStartup),
			app.WithSchedulerLogger(logger),
		)

		return scheduler.Start(ctx)
	})
}
