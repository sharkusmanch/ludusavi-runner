// Package cli provides the command-line interface.
package cli

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/sharkusmanch/ludusavi-runner/internal/config"
	"github.com/sharkusmanch/ludusavi-runner/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	cfgFile  string
	dryRun   bool
	logLevel string
)

// NewRootCmd creates the root command.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ludusavi-runner",
		Short: "Automated Ludusavi game save backup service",
		Long: `ludusavi-runner is a service that automates Ludusavi game save backups
and exports metrics to Prometheus via Pushgateway.

It can run as a one-shot backup, a foreground service, or as a system service.`,
		Version: version.Get().String(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig()
		},
		SilenceUsage: true,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "simulate operations without running ludusavi")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "log level (debug, info, warn, error)")

	// Bind flags to viper
	_ = viper.BindPFlag("dry_run", rootCmd.PersistentFlags().Lookup("dry-run"))
	_ = viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))

	// Add subcommands
	rootCmd.AddCommand(NewRunCmd())
	rootCmd.AddCommand(NewServeCmd())
	rootCmd.AddCommand(NewValidateCmd())
	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewInstallCmd())
	rootCmd.AddCommand(NewUninstallCmd())
	rootCmd.AddCommand(NewStartCmd())
	rootCmd.AddCommand(NewStopCmd())
	rootCmd.AddCommand(NewStatusCmd())

	return rootCmd
}

// Execute runs the root command.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

// initConfig initializes the configuration.
func initConfig() error {
	// Set up basic logging to stderr initially
	// Full logging setup happens in setupLogging after config is loaded
	level := slog.LevelInfo
	if logLevel != "" {
		switch strings.ToLower(logLevel) {
		case "debug":
			level = slog.LevelDebug
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))

	return nil
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

	// CLI flag overrides config
	if logLevel != "" {
		switch strings.ToLower(logLevel) {
		case "debug":
			level = slog.LevelDebug
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}

	// Determine output destination
	var output io.Writer = os.Stderr
	if cfg.Log.Output != "" {
		// Ensure directory exists
		dir := filepath.Dir(cfg.Log.Output)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, err
		}

		// Use lumberjack for log rotation
		output = &lumberjack.Logger{
			Filename:   cfg.Log.Output,
			MaxSize:    cfg.Log.MaxSizeMB,
			MaxBackups: 3,
			MaxAge:     28, // days
			Compress:   true,
		}
	}

	handler := slog.NewTextHandler(output, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger, nil
}

// loadConfig loads the application configuration.
func loadConfig() (*config.Config, error) {
	loader := config.NewLoader()

	if cfgFile != "" {
		loader = loader.WithConfigPath(cfgFile)
	}

	// Apply CLI flag overrides
	if dryRun {
		loader.Set("dry_run", true)
	}
	if logLevel != "" {
		loader.Set("log.level", logLevel)
	}

	return loader.Load()
}
