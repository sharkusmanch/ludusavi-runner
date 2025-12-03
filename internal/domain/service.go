package domain

import "context"

// ServiceState represents the state of a system service.
type ServiceState string

const (
	// ServiceStateUnknown indicates the state cannot be determined.
	ServiceStateUnknown ServiceState = "unknown"
	// ServiceStateStopped indicates the service is stopped.
	ServiceStateStopped ServiceState = "stopped"
	// ServiceStateStarting indicates the service is starting.
	ServiceStateStarting ServiceState = "starting"
	// ServiceStateRunning indicates the service is running.
	ServiceStateRunning ServiceState = "running"
	// ServiceStateStopping indicates the service is stopping.
	ServiceStateStopping ServiceState = "stopping"
	// ServiceStateNotInstalled indicates the service is not installed.
	ServiceStateNotInstalled ServiceState = "not_installed"
)

// String returns the string representation of the service state.
func (s ServiceState) String() string {
	return string(s)
}

// ServiceStatus contains information about the service status.
type ServiceStatus struct {
	// State is the current service state.
	State ServiceState `json:"state"`

	// PID is the process ID if running.
	PID int `json:"pid,omitempty"`

	// StartTime is when the service started, if running.
	StartTime string `json:"start_time,omitempty"`

	// Message provides additional status information.
	Message string `json:"message,omitempty"`
}

// InstallOptions contains options for service installation.
type InstallOptions struct {
	// Username is the account to run the service as.
	Username string

	// Password is the password for the account.
	Password string

	// ConfigPath is the path to the config file.
	ConfigPath string

	// AutoStart enables automatic service start on boot.
	AutoStart bool
}

// ServiceManager defines the interface for managing system services.
// Implementations are platform-specific (Windows, systemd, launchd).
type ServiceManager interface {
	// Install installs the service.
	Install(ctx context.Context, opts InstallOptions) error

	// Uninstall removes the service.
	Uninstall(ctx context.Context) error

	// Start starts the service.
	Start(ctx context.Context) error

	// Stop stops the service.
	Stop(ctx context.Context) error

	// Status returns the current service status.
	Status(ctx context.Context) (*ServiceStatus, error)

	// IsSupported returns true if this service manager is supported on the current platform.
	IsSupported() bool
}
