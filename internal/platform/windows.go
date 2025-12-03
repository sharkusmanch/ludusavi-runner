//go:build windows

package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	serviceName        = "LudusaviRunner"
	serviceDisplayName = "Ludusavi Runner"
	serviceDescription = "Automated Ludusavi game save backup service"
)

// WindowsServiceManager manages Windows services.
type WindowsServiceManager struct{}

// NewServiceManager creates a new service manager for the current platform.
func NewServiceManager() ServiceManager {
	return &WindowsServiceManager{}
}

// IsSupported returns true on Windows.
func (w *WindowsServiceManager) IsSupported() bool {
	return true
}

// Install installs the Windows service.
func (w *WindowsServiceManager) Install(ctx context.Context, opts InstallOptions) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Make path absolute
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Check if service already exists
	s, err := m.OpenService(serviceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", serviceName)
	}

	// Build service arguments
	args := []string{"serve"}
	if opts.ConfigPath != "" {
		args = append(args, "--config", opts.ConfigPath)
	}

	// Build full command line
	binPath := fmt.Sprintf(`"%s" %s`, exePath, strings.Join(args, " "))

	// Determine start type
	startType := uint32(mgr.StartManual)
	if opts.AutoStart {
		startType = uint32(mgr.StartAutomatic)
	}

	// Create service config
	config := mgr.Config{
		DisplayName:      serviceDisplayName,
		Description:      serviceDescription,
		StartType:        startType,
		ServiceStartName: opts.Username,
		Password:         opts.Password,
	}

	s, err = m.CreateService(serviceName, exePath, config, args...)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	// Set recovery options (restart on failure)
	recoveryActions := []mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
	}
	err = s.SetRecoveryActions(recoveryActions, 86400) // Reset after 1 day
	if err != nil {
		// Non-fatal, just log
		fmt.Printf("Warning: failed to set recovery actions: %v\n", err)
	}

	fmt.Printf("Service installed: %s\n", binPath)
	return nil
}

// Uninstall removes the Windows service.
func (w *WindowsServiceManager) Uninstall(ctx context.Context) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s not found: %w", serviceName, err)
	}
	defer s.Close()

	// Stop service if running
	status, err := s.Query()
	if err == nil && status.State != svc.Stopped {
		_, err = s.Control(svc.Stop)
		if err != nil {
			fmt.Printf("Warning: failed to stop service: %v\n", err)
		} else {
			// Wait for stop
			for i := 0; i < 30; i++ {
				status, err = s.Query()
				if err != nil || status.State == svc.Stopped {
					break
				}
				time.Sleep(time.Second)
			}
		}
	}

	err = s.Delete()
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	return nil
}

// Start starts the Windows service.
func (w *WindowsServiceManager) Start(ctx context.Context) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s not found: %w", serviceName, err)
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// Stop stops the Windows service.
func (w *WindowsServiceManager) Stop(ctx context.Context) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s not found: %w", serviceName, err)
	}
	defer s.Close()

	status, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Wait for service to stop
	timeout := time.Now().Add(30 * time.Second)
	for status.State != svc.Stopped {
		if time.Now().After(timeout) {
			return fmt.Errorf("timeout waiting for service to stop")
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("failed to query service status: %w", err)
		}
	}

	return nil
}

// Status returns the current service status.
func (w *WindowsServiceManager) Status(ctx context.Context) (*ServiceStatus, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return &ServiceStatus{
			State:   ServiceStateNotInstalled,
			Message: "Service is not installed",
		}, nil
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return nil, fmt.Errorf("failed to query service status: %w", err)
	}

	var state ServiceState
	switch status.State {
	case svc.Stopped:
		state = ServiceStateStopped
	case svc.StartPending:
		state = ServiceStateStarting
	case svc.Running:
		state = ServiceStateRunning
	case svc.StopPending:
		state = ServiceStateStopping
	default:
		state = ServiceStateUnknown
	}

	return &ServiceStatus{
		State: state,
		PID:   int(status.ProcessId),
	}, nil
}

// RunAsService runs the application as a Windows service.
// This should be called from main() when running as a service.
func RunAsService(handler func(ctx context.Context) error) error {
	return svc.Run(serviceName, &windowsService{handler: handler})
}

// IsRunningAsService returns true if running as a Windows service.
func IsRunningAsService() bool {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return false
	}
	return isService
}

// windowsService implements svc.Handler.
type windowsService struct {
	handler func(ctx context.Context) error
}

func (ws *windowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run the handler in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- ws.handler(ctx)
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for {
		select {
		case err := <-errCh:
			if err != nil {
				// Log error (can't use slog here easily)
				return true, 1
			}
			return false, 0

		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus

			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				// Wait for handler to finish
				<-errCh
				return false, 0
			}
		}
	}
}

// getServicePID gets the PID of a running service using sc.exe
// This is a fallback if the mgr API doesn't provide it.
func getServicePID(serviceName string) int {
	cmd := exec.Command("sc", "queryex", serviceName)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Parse output for PID
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "PID") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				var pid int
				fmt.Sscanf(parts[len(parts)-1], "%d", &pid)
				return pid
			}
		}
	}
	return 0
}
