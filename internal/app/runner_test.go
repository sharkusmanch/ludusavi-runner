package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sharkusmanch/ludusavi-runner/internal/config"
	"github.com/sharkusmanch/ludusavi-runner/internal/domain"
	"github.com/sharkusmanch/ludusavi-runner/internal/executor"
	"github.com/sharkusmanch/ludusavi-runner/internal/metrics"
	"github.com/sharkusmanch/ludusavi-runner/internal/notify"
)

func testConfig() *config.Config {
	return &config.Config{
		Interval:        20 * time.Minute,
		BackupOnStartup: true,
		PushgatewayURL:  "http://localhost:9091",
		Retry: config.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 5 * time.Second,
			MaxDelay:     30 * time.Second,
		},
		Apprise: config.AppriseConfig{
			Enabled: true,
			URL:     "http://localhost:8000",
			Key:     "test",
			Notify:  config.NotifyError,
		},
		Log: config.LogConfig{
			Level:     "info",
			MaxSizeMB: 10,
		},
	}
}

func TestRunner_Run_Success(t *testing.T) {
	cfg := testConfig()

	mockExecutor := &executor.MockExecutor{
		CloudUploadFunc: func(ctx context.Context, opts domain.UploadOptions) (*domain.BackupResult, error) {
			result := domain.NewBackupResult(domain.OperationCloudUpload)
			result.Stats = domain.BackupStats{TotalGames: 50, ProcessedGames: 50}
			result.Complete(true, nil)
			return result, nil
		},
		BackupFunc: func(ctx context.Context, opts domain.BackupOptions) (*domain.BackupResult, error) {
			result := domain.NewBackupResult(domain.OperationBackup)
			result.Stats = domain.BackupStats{
				TotalGames:     100,
				ProcessedGames: 95,
				NewGames:       5,
				ChangedGames:   10,
			}
			result.Complete(true, nil)
			return result, nil
		},
	}

	mockMetrics := &metrics.MockPusher{}
	mockNotifier := &notify.MockNotifier{}

	runner := NewRunner(cfg,
		WithExecutor(mockExecutor),
		WithMetricsPusher(mockMetrics),
		WithNotifier(mockNotifier),
	)

	result, err := runner.Run(context.Background())

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotNil(t, result.CloudUpload)
	assert.NotNil(t, result.Backup)
	assert.True(t, result.CloudUpload.Success)
	assert.True(t, result.Backup.Success)
	assert.Len(t, mockMetrics.PushedMetrics, 1)
	// No notification on success with NotifyError config
	assert.Len(t, mockNotifier.Notifications, 0)
}

func TestRunner_Run_BackupFailure(t *testing.T) {
	cfg := testConfig()

	mockExecutor := &executor.MockExecutor{
		CloudUploadFunc: func(ctx context.Context, opts domain.UploadOptions) (*domain.BackupResult, error) {
			result := domain.NewBackupResult(domain.OperationCloudUpload)
			result.Complete(true, nil)
			return result, nil
		},
		BackupFunc: func(ctx context.Context, opts domain.BackupOptions) (*domain.BackupResult, error) {
			result := domain.NewBackupResult(domain.OperationBackup)
			result.Complete(false, errors.New("backup failed"))
			return result, nil
		},
	}

	mockMetrics := &metrics.MockPusher{}
	mockNotifier := &notify.MockNotifier{}

	runner := NewRunner(cfg,
		WithExecutor(mockExecutor),
		WithMetricsPusher(mockMetrics),
		WithNotifier(mockNotifier),
	)

	result, err := runner.Run(context.Background())

	require.NoError(t, err) // Run doesn't return error, result contains success state
	assert.False(t, result.Success)
	assert.True(t, result.CloudUpload.Success)
	assert.False(t, result.Backup.Success)
	// Should send notification on failure
	assert.Len(t, mockNotifier.Notifications, 1)
	assert.Equal(t, domain.NotificationLevelError, mockNotifier.Notifications[0].Level)
}

func TestRunner_Run_DryRun(t *testing.T) {
	cfg := testConfig()
	cfg.DryRun = true

	callCount := 0
	mockExecutor := &executor.MockExecutor{
		CloudUploadFunc: func(ctx context.Context, opts domain.UploadOptions) (*domain.BackupResult, error) {
			callCount++
			return nil, errors.New("should not be called")
		},
		BackupFunc: func(ctx context.Context, opts domain.BackupOptions) (*domain.BackupResult, error) {
			callCount++
			return nil, errors.New("should not be called")
		},
	}

	mockMetrics := &metrics.MockPusher{}

	runner := NewRunner(cfg,
		WithExecutor(mockExecutor),
		WithMetricsPusher(mockMetrics),
	)

	result, err := runner.Run(context.Background())

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.DryRun)
	// Executor should not be called in dry run
	assert.Equal(t, 0, callCount)
	// Metrics should still be pushed
	assert.Len(t, mockMetrics.PushedMetrics, 1)
}

func TestRunner_Run_NotifyAlways(t *testing.T) {
	cfg := testConfig()
	cfg.Apprise.Notify = config.NotifyAlways

	mockExecutor := &executor.MockExecutor{}
	mockNotifier := &notify.MockNotifier{}

	runner := NewRunner(cfg,
		WithExecutor(mockExecutor),
		WithNotifier(mockNotifier),
	)

	result, err := runner.Run(context.Background())

	require.NoError(t, err)
	assert.True(t, result.Success)
	// Should send notification even on success with NotifyAlways
	assert.Len(t, mockNotifier.Notifications, 1)
	assert.Equal(t, domain.NotificationLevelInfo, mockNotifier.Notifications[0].Level)
}

func TestRunner_Run_NoExecutor(t *testing.T) {
	cfg := testConfig()

	mockMetrics := &metrics.MockPusher{}

	runner := NewRunner(cfg,
		WithMetricsPusher(mockMetrics),
	)

	result, err := runner.Run(context.Background())

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Nil(t, result.CloudUpload)
	assert.Nil(t, result.Backup)
}

func TestRunner_BuildSuccessMessage(t *testing.T) {
	cfg := testConfig()
	runner := NewRunner(cfg)
	runner.hostname = "test-host"

	result := &domain.RunResult{
		Success:  true,
		Duration: 5 * time.Second,
		Backup: &domain.BackupResult{
			Success: true,
			Stats: domain.BackupStats{
				TotalGames:     100,
				ProcessedGames: 95,
				NewGames:       5,
				ChangedGames:   10,
			},
		},
	}

	msg := runner.buildSuccessMessage(result)

	assert.Contains(t, msg, "test-host")
	assert.Contains(t, msg, "100 total")
	assert.Contains(t, msg, "95 processed")
	assert.Contains(t, msg, "5 new")
	assert.Contains(t, msg, "10 updated")
}

func TestRunner_BuildErrorMessage(t *testing.T) {
	cfg := testConfig()
	runner := NewRunner(cfg)
	runner.hostname = "test-host"

	result := &domain.RunResult{
		Success: false,
		Backup: &domain.BackupResult{
			Success: false,
			Error:   "disk full",
		},
		Errors: []string{"additional error"},
	}

	msg := runner.buildErrorMessage(result)

	assert.Contains(t, msg, "test-host")
	assert.Contains(t, msg, "disk full")
	assert.Contains(t, msg, "additional error")
}
