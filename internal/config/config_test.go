package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotifyLevel_IsValid(t *testing.T) {
	tests := []struct {
		level NotifyLevel
		want  bool
	}{
		{NotifyError, true},
		{NotifyWarning, true},
		{NotifyAlways, true},
		{NotifyLevel("invalid"), false},
		{NotifyLevel(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.level.IsValid())
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	validConfig := func() *Config {
		return &Config{
			Interval:        20 * time.Minute,
			BackupOnStartup: true,
			Retry: RetryConfig{
				MaxAttempts:  3,
				InitialDelay: 5 * time.Second,
				MaxDelay:     30 * time.Second,
			},
			Metrics: MetricsConfig{
				Enabled:        true,
				PushgatewayURL: "http://pushgateway:9091",
			},
			Apprise: AppriseConfig{
				Enabled: true,
				URL:     "http://localhost:8000",
				Key:     "ludusavi",
				Notify:  NotifyError,
			},
			Log: LogConfig{
				Level:     "info",
				MaxSizeMB: 10,
			},
		}
	}

	t.Run("valid config", func(t *testing.T) {
		cfg := validConfig()
		assert.NoError(t, cfg.Validate())
	})

	t.Run("interval too short", func(t *testing.T) {
		cfg := validConfig()
		cfg.Interval = 30 * time.Second
		assert.ErrorContains(t, cfg.Validate(), "interval must be at least 1 minute")
	})

	t.Run("empty pushgateway URL when metrics enabled", func(t *testing.T) {
		cfg := validConfig()
		cfg.Metrics.Enabled = true
		cfg.Metrics.PushgatewayURL = ""
		assert.ErrorContains(t, cfg.Validate(), "metrics.pushgateway_url is required when metrics is enabled")
	})

	t.Run("metrics disabled skips validation", func(t *testing.T) {
		cfg := validConfig()
		cfg.Metrics.Enabled = false
		cfg.Metrics.PushgatewayURL = ""
		assert.NoError(t, cfg.Validate())
	})

	t.Run("retry max_attempts less than 1", func(t *testing.T) {
		cfg := validConfig()
		cfg.Retry.MaxAttempts = 0
		assert.ErrorContains(t, cfg.Validate(), "retry.max_attempts must be at least 1")
	})

	t.Run("retry max_delay less than initial_delay", func(t *testing.T) {
		cfg := validConfig()
		cfg.Retry.MaxDelay = 1 * time.Second
		cfg.Retry.InitialDelay = 5 * time.Second
		assert.ErrorContains(t, cfg.Validate(), "retry.max_delay must be >= retry.initial_delay")
	})

	t.Run("apprise enabled without URL", func(t *testing.T) {
		cfg := validConfig()
		cfg.Apprise.Enabled = true
		cfg.Apprise.URL = ""
		assert.ErrorContains(t, cfg.Validate(), "apprise.url is required")
	})

	t.Run("apprise enabled without key", func(t *testing.T) {
		cfg := validConfig()
		cfg.Apprise.Enabled = true
		cfg.Apprise.Key = ""
		assert.ErrorContains(t, cfg.Validate(), "apprise.key is required")
	})

	t.Run("invalid apprise notify level", func(t *testing.T) {
		cfg := validConfig()
		cfg.Apprise.Notify = NotifyLevel("invalid")
		assert.ErrorContains(t, cfg.Validate(), "apprise.notify must be one of")
	})

	t.Run("apprise disabled skips validation", func(t *testing.T) {
		cfg := validConfig()
		cfg.Apprise.Enabled = false
		cfg.Apprise.URL = ""
		cfg.Apprise.Key = ""
		assert.NoError(t, cfg.Validate())
	})

	t.Run("invalid log level", func(t *testing.T) {
		cfg := validConfig()
		cfg.Log.Level = "invalid"
		assert.ErrorContains(t, cfg.Validate(), "log.level must be one of")
	})

	t.Run("log max_size_mb less than 1", func(t *testing.T) {
		cfg := validConfig()
		cfg.Log.MaxSizeMB = 0
		assert.ErrorContains(t, cfg.Validate(), "log.max_size_mb must be at least 1")
	})

	t.Run("non-existent ludusavi path", func(t *testing.T) {
		cfg := validConfig()
		cfg.LudusaviPath = "/non/existent/path"
		assert.ErrorContains(t, cfg.Validate(), "ludusavi_path does not exist")
	})
}

func TestLoader_Load_Defaults(t *testing.T) {
	loader := NewLoader()
	cfg, err := loader.Load()
	require.NoError(t, err)

	assert.Equal(t, DefaultInterval, cfg.Interval)
	assert.Equal(t, DefaultBackupOnStartup, cfg.BackupOnStartup)
	assert.Equal(t, DefaultMetricsEnabled, cfg.Metrics.Enabled)
	assert.Equal(t, DefaultMetricsPushgatewayURL, cfg.Metrics.PushgatewayURL)
	assert.Equal(t, DefaultRetryMaxAttempts, cfg.Retry.MaxAttempts)
	assert.Equal(t, DefaultRetryInitialDelay, cfg.Retry.InitialDelay)
	assert.Equal(t, DefaultRetryMaxDelay, cfg.Retry.MaxDelay)
	assert.Equal(t, DefaultAppriseEnabled, cfg.Apprise.Enabled)
	assert.Equal(t, DefaultAppriseURL, cfg.Apprise.URL)
	assert.Equal(t, DefaultAppriseKey, cfg.Apprise.Key)
	assert.Equal(t, DefaultAppriseNotify, cfg.Apprise.Notify)
	assert.Equal(t, DefaultLogLevel, cfg.Log.Level)
	assert.Equal(t, DefaultLogMaxSizeMB, cfg.Log.MaxSizeMB)
}

func TestLoader_Load_FromFile(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := `
interval = "30m"
backup_on_startup = false

[retry]
max_attempts = 5
initial_delay = "10s"
max_delay = "60s"

[metrics]
enabled = true
pushgateway_url = "http://custom-pushgateway:9091"

[apprise]
enabled = false
url = "http://apprise:8000"
key = "test"
notify = "always"

[log]
level = "debug"
max_size_mb = 20
`
	err := os.WriteFile(configPath, []byte(content), 0600)
	require.NoError(t, err)

	loader := NewLoader().WithConfigPath(configPath)
	cfg, err := loader.Load()
	require.NoError(t, err)

	assert.Equal(t, 30*time.Minute, cfg.Interval)
	assert.False(t, cfg.BackupOnStartup)
	assert.True(t, cfg.Metrics.Enabled)
	assert.Equal(t, "http://custom-pushgateway:9091", cfg.Metrics.PushgatewayURL)
	assert.Equal(t, 5, cfg.Retry.MaxAttempts)
	assert.Equal(t, 10*time.Second, cfg.Retry.InitialDelay)
	assert.Equal(t, 60*time.Second, cfg.Retry.MaxDelay)
	assert.False(t, cfg.Apprise.Enabled)
	assert.Equal(t, "http://apprise:8000", cfg.Apprise.URL)
	assert.Equal(t, "test", cfg.Apprise.Key)
	assert.Equal(t, NotifyAlways, cfg.Apprise.Notify)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, 20, cfg.Log.MaxSizeMB)
}

func TestLoader_Load_EnvOverrides(t *testing.T) {
	// Set environment variables
	t.Setenv("LUDUSAVI_RUNNER_INTERVAL", "45m")
	t.Setenv("LUDUSAVI_RUNNER_BACKUP_ON_STARTUP", "false")
	t.Setenv("LUDUSAVI_RUNNER_LOG_LEVEL", "debug")

	loader := NewLoader()
	cfg, err := loader.Load()
	require.NoError(t, err)

	assert.Equal(t, 45*time.Minute, cfg.Interval)
	assert.False(t, cfg.BackupOnStartup)
	assert.Equal(t, "debug", cfg.Log.Level)
}

func TestLoader_Set(t *testing.T) {
	loader := NewLoader()
	loader.Set("interval", "60m")
	loader.Set("log.level", "error")

	cfg, err := loader.Load()
	require.NoError(t, err)

	assert.Equal(t, 60*time.Minute, cfg.Interval)
	assert.Equal(t, "error", cfg.Log.Level)
}

func TestWriteExampleConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.toml")

	err := WriteExampleConfig(configPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Verify it can be loaded
	loader := NewLoader().WithConfigPath(configPath)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Should have default values
	assert.Equal(t, DefaultInterval, cfg.Interval)
}

func TestDefaultConfigDir(t *testing.T) {
	dir, err := DefaultConfigDir()
	require.NoError(t, err)
	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, AppName)
}

func TestDefaultConfigPath(t *testing.T) {
	path, err := DefaultConfigPath()
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, ConfigFileName)
}
