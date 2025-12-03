// Package platform provides platform-specific service management.
package platform

import (
	"context"

	"github.com/sharkusmanch/ludusavi-runner/internal/domain"
)

// InstallOptions contains options for service installation.
type InstallOptions = domain.InstallOptions

// ServiceStatus contains service status information.
type ServiceStatus = domain.ServiceStatus

// ServiceState represents service state.
type ServiceState = domain.ServiceState

// Service state constants.
const (
	ServiceStateUnknown      = domain.ServiceStateUnknown
	ServiceStateStopped      = domain.ServiceStateStopped
	ServiceStateStarting     = domain.ServiceStateStarting
	ServiceStateRunning      = domain.ServiceStateRunning
	ServiceStateStopping     = domain.ServiceStateStopping
	ServiceStateNotInstalled = domain.ServiceStateNotInstalled
)

// ServiceManager defines the interface for managing system services.
type ServiceManager interface {
	Install(ctx context.Context, opts InstallOptions) error
	Uninstall(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Status(ctx context.Context) (*ServiceStatus, error)
	IsSupported() bool
}
