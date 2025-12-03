package cli

import (
	"fmt"

	"github.com/sharkusmanch/ludusavi-runner/internal/platform"
	"github.com/spf13/cobra"
)

var (
	installUsername string
)

// NewInstallCmd creates the install command.
func NewInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install as a system service",
		Long: `Install ludusavi-runner as a system service.

On Windows, this installs a Windows Service.
On Linux, this would install a systemd unit (not yet implemented).
On macOS, this would install a launchd plist (not yet implemented).`,
		RunE: runInstall,
	}

	cmd.Flags().StringVar(&installUsername, "username", "", "username to run the service as (Windows)")

	return cmd
}

func runInstall(cmd *cobra.Command, args []string) error {
	mgr := platform.NewServiceManager()

	if !mgr.IsSupported() {
		return fmt.Errorf("service management is not supported on this platform")
	}

	opts := platform.InstallOptions{
		Username:   installUsername,
		ConfigPath: cfgFile,
		AutoStart:  true,
	}

	if err := mgr.Install(cmd.Context(), opts); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	fmt.Println("Service installed successfully.")
	fmt.Println("Use 'ludusavi-runner start' to start the service.")
	return nil
}

// NewUninstallCmd creates the uninstall command.
func NewUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the system service",
		Long:  `Remove the ludusavi-runner system service.`,
		RunE:  runUninstall,
	}

	return cmd
}

func runUninstall(cmd *cobra.Command, args []string) error {
	mgr := platform.NewServiceManager()

	if !mgr.IsSupported() {
		return fmt.Errorf("service management is not supported on this platform")
	}

	if err := mgr.Uninstall(cmd.Context()); err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}

	fmt.Println("Service uninstalled successfully.")
	return nil
}

// NewStartCmd creates the start command.
func NewStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the installed service",
		Long:  `Start the ludusavi-runner system service.`,
		RunE:  runStart,
	}

	return cmd
}

func runStart(cmd *cobra.Command, args []string) error {
	mgr := platform.NewServiceManager()

	if !mgr.IsSupported() {
		return fmt.Errorf("service management is not supported on this platform")
	}

	if err := mgr.Start(cmd.Context()); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	fmt.Println("Service started.")
	return nil
}

// NewStopCmd creates the stop command.
func NewStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the installed service",
		Long:  `Stop the ludusavi-runner system service.`,
		RunE:  runStop,
	}

	return cmd
}

func runStop(cmd *cobra.Command, args []string) error {
	mgr := platform.NewServiceManager()

	if !mgr.IsSupported() {
		return fmt.Errorf("service management is not supported on this platform")
	}

	if err := mgr.Stop(cmd.Context()); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	fmt.Println("Service stopped.")
	return nil
}

// NewStatusCmd creates the status command.
func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show service status",
		Long:  `Display the current status of the ludusavi-runner system service.`,
		RunE:  runStatus,
	}

	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
	mgr := platform.NewServiceManager()

	if !mgr.IsSupported() {
		return fmt.Errorf("service management is not supported on this platform")
	}

	status, err := mgr.Status(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}

	fmt.Printf("Service Status: %s\n", status.State)
	if status.PID > 0 {
		fmt.Printf("PID: %d\n", status.PID)
	}
	if status.StartTime != "" {
		fmt.Printf("Start Time: %s\n", status.StartTime)
	}
	if status.Message != "" {
		fmt.Printf("Message: %s\n", status.Message)
	}

	return nil
}
