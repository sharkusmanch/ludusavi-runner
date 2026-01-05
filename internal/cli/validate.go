package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/sharkusmanch/ludusavi-runner/internal/config"
	"github.com/sharkusmanch/ludusavi-runner/internal/executor"
	"github.com/sharkusmanch/ludusavi-runner/internal/http"
	"github.com/sharkusmanch/ludusavi-runner/internal/metrics"
	"github.com/sharkusmanch/ludusavi-runner/internal/notify"
	"github.com/spf13/cobra"
)

// NewValidateCmd creates the validate command.
func NewValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration and test connectivity",
		Long: `Validate the configuration file and test connectivity to external services.

This checks:
- Config file syntax
- Ludusavi binary availability
- Pushgateway connectivity
- Apprise server connectivity (if enabled)`,
		RunE: runValidate,
	}

	return cmd
}

func runValidate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	// Load config
	fmt.Println("Configuration:")
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("  ✗ Config file: %v\n", err)
		return err
	}
	fmt.Printf("  ✓ Config file syntax valid\n")

	// Display config values
	configPath, _ := config.DefaultConfigPath()
	if cfgFile != "" {
		configPath = cfgFile
	}
	fmt.Printf("  Config file: %s\n", configPath)
	fmt.Printf("  Interval: %s\n", cfg.Interval)
	fmt.Printf("  Backup on startup: %t\n", cfg.BackupOnStartup)
	if cfg.Metrics.Enabled {
		fmt.Printf("  Metrics: enabled\n")
		fmt.Printf("  Pushgateway URL: %s\n", cfg.Metrics.PushgatewayURL)
	} else {
		fmt.Printf("  Metrics: disabled\n")
	}
	if cfg.Apprise.Enabled {
		fmt.Printf("  Notifications: enabled\n")
		fmt.Printf("  Apprise URL: %s\n", cfg.Apprise.URL)
		fmt.Printf("  Notification level: %s\n", cfg.Apprise.Notify)
	} else {
		fmt.Printf("  Notifications: disabled\n")
	}
	fmt.Println()

	// Check ludusavi
	fmt.Println("Checks:")
	logger, _ := setupLogging(cfg)
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

	if err := exec.Validate(ctx); err != nil {
		fmt.Printf("  ✗ Ludusavi binary: %v\n", err)
	} else {
		version, _ := exec.Version(ctx)
		fmt.Printf("  ✓ Ludusavi binary found: %s\n", version)
	}

	// Create HTTP client
	httpClient := http.NewClient(
		http.WithRetryConfig(http.RetryConfig{
			MaxAttempts:  1, // No retries for validation
			InitialDelay: time.Second,
			MaxDelay:     time.Second,
		}),
		http.WithLogger(logger),
	)

	// Check pushgateway if enabled
	if cfg.Metrics.Enabled {
		pushgatewayClient := metrics.NewPushgatewayClient(
			cfg.Metrics.PushgatewayURL,
			metrics.WithHTTPClient(httpClient),
			metrics.WithLogger(logger),
		)

		if err := pushgatewayClient.Validate(ctx); err != nil {
			fmt.Printf("  ✗ Pushgateway: %v\n", err)
		} else {
			fmt.Printf("  ✓ Pushgateway reachable\n")
		}
	}

	// Check apprise if enabled
	if cfg.Apprise.Enabled {
		appriseClient := notify.NewAppriseClient(
			cfg.Apprise.URL,
			cfg.Apprise.Key,
			notify.WithHTTPClient(httpClient),
			notify.WithLogger(logger),
		)

		if err := appriseClient.Validate(ctx); err != nil {
			fmt.Printf("  ✗ Apprise server: %v\n", err)
		} else {
			fmt.Printf("  ✓ Apprise server reachable\n")
		}
	}

	fmt.Println()
	fmt.Println("Validation complete.")
	return nil
}
