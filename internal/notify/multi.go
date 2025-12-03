package notify

import (
	"context"
	"errors"
	"log/slog"

	"github.com/sharkusmanch/ludusavi-runner/internal/domain"
)

// MultiNotifier sends notifications to multiple notifiers.
type MultiNotifier struct {
	notifiers []domain.Notifier
	logger    *slog.Logger
}

// NewMultiNotifier creates a new MultiNotifier.
func NewMultiNotifier(notifiers ...domain.Notifier) *MultiNotifier {
	return &MultiNotifier{
		notifiers: notifiers,
		logger:    slog.Default(),
	}
}

// Notify sends a notification to all configured notifiers.
// Returns an error if any notifier fails, but attempts all notifiers.
func (m *MultiNotifier) Notify(ctx context.Context, notification *domain.Notification) error {
	var errs []error

	for _, notifier := range m.notifiers {
		if err := notifier.Notify(ctx, notification); err != nil {
			m.logger.Warn("notifier failed", "error", err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Validate validates all configured notifiers.
func (m *MultiNotifier) Validate(ctx context.Context) error {
	var errs []error

	for _, notifier := range m.notifiers {
		if err := notifier.Validate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Ensure MultiNotifier implements domain.Notifier.
var _ domain.Notifier = (*MultiNotifier)(nil)
