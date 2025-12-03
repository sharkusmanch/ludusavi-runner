package domain

import (
	"context"
	"time"
)

// Metrics contains all metrics to be pushed.
type Metrics struct {
	// Timestamp when metrics were collected.
	Timestamp time.Time

	// Hostname of the machine.
	Hostname string

	// ServiceUp indicates if the service is running.
	ServiceUp bool

	// Version information.
	Version   string
	GoVersion string

	// Results from backup operations.
	Results []*BackupResult
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(hostname string) *Metrics {
	return &Metrics{
		Timestamp: time.Now(),
		Hostname:  hostname,
		ServiceUp: true,
		Results:   make([]*BackupResult, 0),
	}
}

// AddResult adds a backup result to the metrics.
func (m *Metrics) AddResult(result *BackupResult) {
	if result != nil {
		m.Results = append(m.Results, result)
	}
}

// MetricsPusher defines the interface for pushing metrics to a remote endpoint.
type MetricsPusher interface {
	// Push sends metrics to the remote endpoint.
	Push(ctx context.Context, metrics *Metrics) error

	// Validate checks if the pusher is properly configured.
	Validate(ctx context.Context) error
}
