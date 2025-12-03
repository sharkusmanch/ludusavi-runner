package domain

import "context"

// NotificationLevel represents the severity of a notification.
type NotificationLevel string

const (
	// NotificationLevelInfo is for informational messages.
	NotificationLevelInfo NotificationLevel = "info"
	// NotificationLevelWarning is for warning messages.
	NotificationLevelWarning NotificationLevel = "warning"
	// NotificationLevelError is for error messages.
	NotificationLevelError NotificationLevel = "error"
)

// Notification represents a notification to be sent.
type Notification struct {
	// Title is the notification title.
	Title string `json:"title"`

	// Body is the notification body/message.
	Body string `json:"body"`

	// Level is the severity level.
	Level NotificationLevel `json:"level"`
}

// NewNotification creates a new notification.
func NewNotification(title, body string, level NotificationLevel) *Notification {
	return &Notification{
		Title: title,
		Body:  body,
		Level: level,
	}
}

// InfoNotification creates an info-level notification.
func InfoNotification(title, body string) *Notification {
	return NewNotification(title, body, NotificationLevelInfo)
}

// WarningNotification creates a warning-level notification.
func WarningNotification(title, body string) *Notification {
	return NewNotification(title, body, NotificationLevelWarning)
}

// ErrorNotification creates an error-level notification.
func ErrorNotification(title, body string) *Notification {
	return NewNotification(title, body, NotificationLevelError)
}

// Notifier defines the interface for sending notifications.
type Notifier interface {
	// Notify sends a notification.
	Notify(ctx context.Context, notification *Notification) error

	// Validate checks if the notifier is properly configured.
	Validate(ctx context.Context) error
}

// NopNotifier is a no-op notifier that does nothing.
type NopNotifier struct{}

// Notify does nothing.
func (n *NopNotifier) Notify(_ context.Context, _ *Notification) error {
	return nil
}

// Validate always returns nil.
func (n *NopNotifier) Validate(_ context.Context) error {
	return nil
}
