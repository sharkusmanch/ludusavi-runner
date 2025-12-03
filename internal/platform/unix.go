//go:build !windows

package platform

import (
	"context"
	"fmt"
)

// UnixServiceManager is a stub service manager for non-Windows platforms.
type UnixServiceManager struct{}

// NewServiceManager creates a new service manager for the current platform.
func NewServiceManager() ServiceManager {
	return &UnixServiceManager{}
}

// IsSupported returns false on non-Windows platforms (for now).
func (u *UnixServiceManager) IsSupported() bool {
	return false
}

// Install is not implemented on non-Windows platforms.
func (u *UnixServiceManager) Install(ctx context.Context, opts InstallOptions) error {
	return fmt.Errorf("service installation is not yet supported on this platform")
}

// Uninstall is not implemented on non-Windows platforms.
func (u *UnixServiceManager) Uninstall(ctx context.Context) error {
	return fmt.Errorf("service uninstallation is not yet supported on this platform")
}

// Start is not implemented on non-Windows platforms.
func (u *UnixServiceManager) Start(ctx context.Context) error {
	return fmt.Errorf("service start is not yet supported on this platform")
}

// Stop is not implemented on non-Windows platforms.
func (u *UnixServiceManager) Stop(ctx context.Context) error {
	return fmt.Errorf("service stop is not yet supported on this platform")
}

// Status is not implemented on non-Windows platforms.
func (u *UnixServiceManager) Status(ctx context.Context) (*ServiceStatus, error) {
	return &ServiceStatus{
		State:   ServiceStateUnknown,
		Message: "Service management is not yet supported on this platform",
	}, nil
}

// RunAsService is not implemented on non-Windows platforms.
func RunAsService(handler func(ctx context.Context) error) error {
	return fmt.Errorf("running as service is not yet supported on this platform")
}

// IsRunningAsService returns false on non-Windows platforms.
func IsRunningAsService() bool {
	return false
}
