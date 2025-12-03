package metrics

import (
	"context"

	"github.com/sharkusmanch/ludusavi-runner/internal/domain"
)

// MockPusher is a mock implementation of domain.MetricsPusher for testing.
type MockPusher struct {
	PushFunc     func(ctx context.Context, metrics *domain.Metrics) error
	ValidateFunc func(ctx context.Context) error

	// PushedMetrics stores all metrics that have been pushed.
	PushedMetrics []*domain.Metrics
}

// Push calls the mock PushFunc and stores the metrics.
func (m *MockPusher) Push(ctx context.Context, metrics *domain.Metrics) error {
	m.PushedMetrics = append(m.PushedMetrics, metrics)
	if m.PushFunc != nil {
		return m.PushFunc(ctx, metrics)
	}
	return nil
}

// Validate calls the mock ValidateFunc.
func (m *MockPusher) Validate(ctx context.Context) error {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(ctx)
	}
	return nil
}

// Reset clears all stored metrics.
func (m *MockPusher) Reset() {
	m.PushedMetrics = nil
}

// Ensure MockPusher implements domain.MetricsPusher.
var _ domain.MetricsPusher = (*MockPusher)(nil)
