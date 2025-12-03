// Package app provides the core application logic.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/sharkusmanch/ludusavi-runner/internal/config"
	"github.com/sharkusmanch/ludusavi-runner/internal/domain"
)

// Runner orchestrates backup operations.
type Runner struct {
	executor      domain.Executor
	metricsPusher domain.MetricsPusher
	notifier      domain.Notifier
	config        *config.Config
	logger        *slog.Logger
	hostname      string
}

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithExecutor sets the executor.
func WithExecutor(e domain.Executor) RunnerOption {
	return func(r *Runner) {
		r.executor = e
	}
}

// WithMetricsPusher sets the metrics pusher.
func WithMetricsPusher(m domain.MetricsPusher) RunnerOption {
	return func(r *Runner) {
		r.metricsPusher = m
	}
}

// WithNotifier sets the notifier.
func WithNotifier(n domain.Notifier) RunnerOption {
	return func(r *Runner) {
		r.notifier = n
	}
}

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) RunnerOption {
	return func(r *Runner) {
		r.logger = l
	}
}

// NewRunner creates a new Runner.
func NewRunner(cfg *config.Config, opts ...RunnerOption) *Runner {
	hostname, _ := os.Hostname()

	r := &Runner{
		config:   cfg,
		logger:   slog.Default(),
		hostname: hostname,
		notifier: &domain.NopNotifier{}, // Default to no-op
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Run executes a single backup cycle.
func (r *Runner) Run(ctx context.Context) (*domain.RunResult, error) {
	result := domain.NewRunResult(r.config.DryRun)

	r.logger.Info("starting backup run", "dry_run", r.config.DryRun)

	// Execute cloud upload first
	if r.executor != nil {
		uploadResult, err := r.runCloudUpload(ctx)
		if err != nil {
			r.logger.Error("cloud upload failed", "error", err)
			result.AddError(err)
		}
		result.CloudUpload = uploadResult

		// Execute local backup
		backupResult, err := r.runBackup(ctx)
		if err != nil {
			r.logger.Error("backup failed", "error", err)
			result.AddError(err)
		}
		result.Backup = backupResult
	}

	result.Complete()

	// Push metrics
	if err := r.pushMetrics(ctx, result); err != nil {
		r.logger.Error("failed to push metrics", "error", err)
		result.AddError(err)
	}

	// Send notifications based on result and config
	if err := r.sendNotifications(ctx, result); err != nil {
		r.logger.Error("failed to send notification", "error", err)
	}

	r.logger.Info("backup run completed",
		"success", result.Success,
		"duration", result.Duration,
	)

	return result, nil
}

// runCloudUpload executes the cloud upload operation.
func (r *Runner) runCloudUpload(ctx context.Context) (*domain.BackupResult, error) {
	r.logger.Debug("starting cloud upload")

	if r.config.DryRun {
		r.logger.Info("dry run: skipping cloud upload")
		result := domain.NewBackupResult(domain.OperationCloudUpload)
		result.Complete(true, nil)
		return result, nil
	}

	result, err := r.executor.CloudUpload(ctx, domain.UploadOptions{Force: true})
	if err != nil {
		return nil, fmt.Errorf("cloud upload error: %w", err)
	}

	if result.Success {
		r.logger.Info("cloud upload completed",
			"games_processed", result.Stats.ProcessedGames,
			"bytes_processed", result.Stats.ProcessedBytes,
			"duration", result.Duration,
		)
	} else {
		r.logger.Warn("cloud upload failed", "error", result.Error)
	}

	return result, nil
}

// runBackup executes the local backup operation.
func (r *Runner) runBackup(ctx context.Context) (*domain.BackupResult, error) {
	r.logger.Debug("starting local backup")

	if r.config.DryRun {
		r.logger.Info("dry run: skipping local backup")
		result := domain.NewBackupResult(domain.OperationBackup)
		result.Complete(true, nil)
		return result, nil
	}

	result, err := r.executor.Backup(ctx, domain.BackupOptions{Force: true})
	if err != nil {
		return nil, fmt.Errorf("backup error: %w", err)
	}

	if result.Success {
		r.logger.Info("local backup completed",
			"games_total", result.Stats.TotalGames,
			"games_processed", result.Stats.ProcessedGames,
			"bytes_processed", result.Stats.ProcessedBytes,
			"games_new", result.Stats.NewGames,
			"games_changed", result.Stats.ChangedGames,
			"duration", result.Duration,
		)
	} else {
		r.logger.Warn("local backup failed", "error", result.Error)
	}

	return result, nil
}

// pushMetrics sends metrics to the metrics pusher.
func (r *Runner) pushMetrics(ctx context.Context, result *domain.RunResult) error {
	if r.metricsPusher == nil {
		return nil
	}

	metrics := domain.NewMetrics(r.hostname)
	metrics.ServiceUp = true

	if result.CloudUpload != nil {
		metrics.AddResult(result.CloudUpload)
	}
	if result.Backup != nil {
		metrics.AddResult(result.Backup)
	}

	return r.metricsPusher.Push(ctx, metrics)
}

// sendNotifications sends notifications based on the result and config.
func (r *Runner) sendNotifications(ctx context.Context, result *domain.RunResult) error {
	if r.notifier == nil {
		return nil
	}

	notifyLevel := r.config.Apprise.Notify

	// Determine if we should notify
	shouldNotify := false
	var notification *domain.Notification

	switch {
	case !result.Success && (notifyLevel == config.NotifyError || notifyLevel == config.NotifyWarning || notifyLevel == config.NotifyAlways):
		shouldNotify = true
		notification = domain.ErrorNotification(
			"Ludusavi Backup Failed",
			r.buildErrorMessage(result),
		)

	case notifyLevel == config.NotifyAlways:
		shouldNotify = true
		notification = domain.InfoNotification(
			"Ludusavi Backup Completed",
			r.buildSuccessMessage(result),
		)
	}

	if !shouldNotify || notification == nil {
		return nil
	}

	return r.notifier.Notify(ctx, notification)
}

// buildErrorMessage builds an error notification message.
func (r *Runner) buildErrorMessage(result *domain.RunResult) string {
	msg := fmt.Sprintf("Backup failed on %s.\n", r.hostname)

	if result.CloudUpload != nil && !result.CloudUpload.Success {
		msg += fmt.Sprintf("Cloud upload error: %s\n", result.CloudUpload.Error)
	}
	if result.Backup != nil && !result.Backup.Success {
		msg += fmt.Sprintf("Backup error: %s\n", result.Backup.Error)
	}

	for _, err := range result.Errors {
		msg += fmt.Sprintf("Error: %s\n", err)
	}

	return msg
}

// buildSuccessMessage builds a success notification message.
func (r *Runner) buildSuccessMessage(result *domain.RunResult) string {
	msg := fmt.Sprintf("Backup completed successfully on %s.\n", r.hostname)

	if result.Backup != nil {
		msg += fmt.Sprintf("Games: %d total, %d processed\n",
			result.Backup.Stats.TotalGames,
			result.Backup.Stats.ProcessedGames,
		)
		if result.Backup.Stats.NewGames > 0 || result.Backup.Stats.ChangedGames > 0 {
			msg += fmt.Sprintf("Changes: %d new, %d updated\n",
				result.Backup.Stats.NewGames,
				result.Backup.Stats.ChangedGames,
			)
		}
	}

	msg += fmt.Sprintf("Duration: %s", result.Duration.Round(100000000)) // Round to 0.1s

	return msg
}
