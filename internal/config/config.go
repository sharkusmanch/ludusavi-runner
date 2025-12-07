package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Interval        time.Duration `mapstructure:"interval"`
	BackupOnStartup bool          `mapstructure:"backup_on_startup"`
	LudusaviPath    string        `mapstructure:"ludusavi_path"`
	DryRun          bool          `mapstructure:"dry_run"`
	Retry           RetryConfig   `mapstructure:"retry"`
	Metrics         MetricsConfig `mapstructure:"metrics"`
	Apprise         AppriseConfig `mapstructure:"apprise"`
	Log             LogConfig     `mapstructure:"log"`
}

// MetricsConfig holds Prometheus metrics configuration.
type MetricsConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	PushgatewayURL string `mapstructure:"pushgateway_url"`
}

// RetryConfig holds HTTP retry configuration.
type RetryConfig struct {
	MaxAttempts  int           `mapstructure:"max_attempts"`
	InitialDelay time.Duration `mapstructure:"initial_delay"`
	MaxDelay     time.Duration `mapstructure:"max_delay"`
}

// AppriseConfig holds Apprise notification configuration.
type AppriseConfig struct {
	Enabled bool        `mapstructure:"enabled"`
	URL     string      `mapstructure:"url"`
	Key     string      `mapstructure:"key"`
	Notify  NotifyLevel `mapstructure:"notify"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level     string `mapstructure:"level"`
	Output    string `mapstructure:"output"`
	MaxSizeMB int    `mapstructure:"max_size_mb"`
}

// Loader handles configuration loading from multiple sources.
type Loader struct {
	v          *viper.Viper
	configPath string
}

// NewLoader creates a new configuration loader.
func NewLoader() *Loader {
	return &Loader{
		v: viper.New(),
	}
}

// WithConfigPath sets a specific config file path.
func (l *Loader) WithConfigPath(path string) *Loader {
	l.configPath = path
	return l
}

// Load reads configuration from all sources and returns the merged config.
// Precedence (highest to lowest): CLI flags > environment > config file > defaults.
func (l *Loader) Load() (*Config, error) {
	l.setDefaults()
	l.setupEnvBindings()

	if err := l.loadConfigFile(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := l.v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set default log path if not specified.
	// This is done after loading because the default path depends on the config directory.
	if cfg.Log.Output == "" {
		logPath, err := DefaultLogPath()
		if err == nil {
			cfg.Log.Output = logPath
		}
		// If we can't determine the default path, leave it empty (will log to stderr)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default values for all configuration options.
func (l *Loader) setDefaults() {
	l.v.SetDefault("interval", DefaultInterval)
	l.v.SetDefault("backup_on_startup", DefaultBackupOnStartup)
	l.v.SetDefault("ludusavi_path", "")
	l.v.SetDefault("dry_run", false)

	l.v.SetDefault("retry.max_attempts", DefaultRetryMaxAttempts)
	l.v.SetDefault("retry.initial_delay", DefaultRetryInitialDelay)
	l.v.SetDefault("retry.max_delay", DefaultRetryMaxDelay)

	l.v.SetDefault("metrics.enabled", DefaultMetricsEnabled)
	l.v.SetDefault("metrics.pushgateway_url", DefaultMetricsPushgatewayURL)

	l.v.SetDefault("apprise.enabled", DefaultAppriseEnabled)
	l.v.SetDefault("apprise.url", DefaultAppriseURL)
	l.v.SetDefault("apprise.key", DefaultAppriseKey)
	l.v.SetDefault("apprise.notify", string(DefaultAppriseNotify))

	l.v.SetDefault("log.level", DefaultLogLevel)
	l.v.SetDefault("log.output", "")
	l.v.SetDefault("log.max_size_mb", DefaultLogMaxSizeMB)
}

// setupEnvBindings configures environment variable bindings.
func (l *Loader) setupEnvBindings() {
	l.v.SetEnvPrefix(EnvPrefix)
	l.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	l.v.AutomaticEnv()
}

// loadConfigFile loads configuration from a file.
func (l *Loader) loadConfigFile() error {
	if l.configPath != "" {
		// Specific config file provided
		l.v.SetConfigFile(l.configPath)
	} else {
		// Look for config in default locations
		configDir, err := DefaultConfigDir()
		if err != nil {
			// Can't determine config dir, proceed without file config
			return nil
		}

		l.v.SetConfigName("config")
		l.v.SetConfigType("toml")
		l.v.AddConfigPath(configDir)
		l.v.AddConfigPath(".")
	}

	if err := l.v.ReadInConfig(); err != nil {
		// Config file not found is not an error - use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	return nil
}

// Set sets a configuration value (for CLI flag overrides).
func (l *Loader) Set(key string, value interface{}) {
	l.v.Set(key, value)
}

// ConfigFileUsed returns the path of the config file used, if any.
func (l *Loader) ConfigFileUsed() string {
	return l.v.ConfigFileUsed()
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Interval < time.Minute {
		return fmt.Errorf("interval must be at least 1 minute, got %s", c.Interval)
	}

	if c.LudusaviPath != "" {
		if _, err := os.Stat(c.LudusaviPath); err != nil {
			return fmt.Errorf("ludusavi_path does not exist: %s", c.LudusaviPath)
		}
	}

	if c.Metrics.Enabled {
		if c.Metrics.PushgatewayURL == "" {
			return fmt.Errorf("metrics.pushgateway_url is required when metrics is enabled")
		}
	}

	if c.Retry.MaxAttempts < 1 {
		return fmt.Errorf("retry.max_attempts must be at least 1")
	}

	if c.Retry.InitialDelay < 0 {
		return fmt.Errorf("retry.initial_delay cannot be negative")
	}

	if c.Retry.MaxDelay < c.Retry.InitialDelay {
		return fmt.Errorf("retry.max_delay must be >= retry.initial_delay")
	}

	if c.Apprise.Enabled {
		if c.Apprise.URL == "" {
			return fmt.Errorf("apprise.url is required when apprise is enabled")
		}
		if c.Apprise.Key == "" {
			return fmt.Errorf("apprise.key is required when apprise is enabled")
		}
		if !c.Apprise.Notify.IsValid() {
			return fmt.Errorf("apprise.notify must be one of: error, warning, always")
		}
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[strings.ToLower(c.Log.Level)] {
		return fmt.Errorf("log.level must be one of: debug, info, warn, error")
	}

	if c.Log.MaxSizeMB < 1 {
		return fmt.Errorf("log.max_size_mb must be at least 1")
	}

	return nil
}

// EnsureConfigDir creates the config directory if it doesn't exist.
func EnsureConfigDir() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	return dir, nil
}

// WriteExampleConfig writes an example config file to the given path.
func WriteExampleConfig(path string) error {
	content := `# Ludusavi Runner Configuration

# Backup schedule interval
interval = "20m"

# Run backup immediately on service start
backup_on_startup = true

# Path to ludusavi binary (auto-detected if empty)
ludusavi_path = ""

# HTTP retry configuration
[retry]
max_attempts = 3
initial_delay = "5s"
max_delay = "30s"

# Prometheus metrics (optional, disabled by default)
[metrics]
enabled = false
pushgateway_url = "http://pushgateway:9091"

# Apprise notifications (optional, disabled by default)
[apprise]
enabled = false
url = "http://localhost:8000"
key = "ludusavi"
# Notification level: "error", "warning", "always"
notify = "error"

# Logging configuration
[log]
# Level: debug, info, warn, error
level = "info"
# Output file path (defaults to ludusavi-runner.log in config directory)
# output = ""
# Max log file size before rotation (MB)
max_size_mb = 10
`
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	return os.WriteFile(path, []byte(content), 0600)
}
