package notify

import (
	"context"

	"github.com/sharkusmanch/ludusavi-runner/internal/domain"
)

// MockNotifier is a mock implementation of domain.Notifier for testing.
type MockNotifier struct {
	NotifyFunc   func(ctx context.Context, notification *domain.Notification) error
	ValidateFunc func(ctx context.Context) error

	// Notifications stores all notifications that have been sent.
	Notifications []*domain.Notification
}

// Notify calls the mock NotifyFunc and stores the notification.
func (m *MockNotifier) Notify(ctx context.Context, notification *domain.Notification) error {
	m.Notifications = append(m.Notifications, notification)
	if m.NotifyFunc != nil {
		return m.NotifyFunc(ctx, notification)
	}
	return nil
}

// Validate calls the mock ValidateFunc.
func (m *MockNotifier) Validate(ctx context.Context) error {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(ctx)
	}
	return nil
}

// Reset clears all stored notifications.
func (m *MockNotifier) Reset() {
	m.Notifications = nil
}

// Ensure MockNotifier implements domain.Notifier.
var _ domain.Notifier = (*MockNotifier)(nil)
