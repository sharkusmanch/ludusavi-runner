// Package main is the entry point for ludusavi-runner.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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

// setupLogging configures logging based on the loaded config.
func setupLogging(cfg *config.Config) (*slog.Logger, error) {
	// Determine log level
	level := slog.LevelInfo
	switch strings.ToLower(cfg.Log.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	// Determine output destination
	var output io.Writer = os.Stderr
	if cfg.Log.Output != "" {
		// Ensure directory exists
		dir := filepath.Dir(cfg.Log.Output)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		file, err := os.OpenFile(cfg.Log.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		output = file
	}

	handler := slog.NewTextHandler(output, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger, nil
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
		logger, err := setupLogging(cfg)
		if err != nil {
			return err
		}

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
		if len(cfg.Env) > 0 {
			execOpts = append(execOpts, executor.WithEnv(cfg.Env))
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
