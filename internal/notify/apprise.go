// Package notify provides implementations for sending notifications.
package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/sharkusmanch/ludusavi-runner/internal/domain"
	"github.com/sharkusmanch/ludusavi-runner/internal/http"
)

const (
	maxBodyLength = 1000
)

// AppriseClient sends notifications via an Apprise server.
type AppriseClient struct {
	url        string
	key        string
	httpClient *http.Client
	logger     *slog.Logger
}

// AppriseOption configures an AppriseClient.
type AppriseOption func(*AppriseClient)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) AppriseOption {
	return func(a *AppriseClient) {
		a.httpClient = client
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) AppriseOption {
	return func(a *AppriseClient) {
		a.logger = logger
	}
}

// NewAppriseClient creates a new AppriseClient.
func NewAppriseClient(url, key string, opts ...AppriseOption) *AppriseClient {
	a := &AppriseClient{
		url:        strings.TrimSuffix(url, "/"),
		key:        key,
		httpClient: http.NewClient(),
		logger:     slog.Default(),
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// appriseRequest represents the JSON body sent to Apprise.
type appriseRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Type  string `json:"type,omitempty"` // info, success, warning, failure
}

// Notify sends a notification via Apprise.
func (a *AppriseClient) Notify(ctx context.Context, notification *domain.Notification) error {
	body := notification.Body
	if len(body) > maxBodyLength {
		body = body[:maxBodyLength-3] + "..."
	}

	req := appriseRequest{
		Title: notification.Title,
		Body:  body,
		Type:  a.mapLevel(notification.Level),
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	notifyURL := fmt.Sprintf("%s/notify/%s", a.url, a.key)

	a.logger.Debug("sending notification via apprise",
		"url", notifyURL,
		"title", notification.Title,
		"level", notification.Level,
	)

	resp, err := a.httpClient.Post(ctx, notifyURL, "application/json", jsonBody)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("apprise returned status %d: %s", resp.StatusCode, string(resp.Body))
	}

	a.logger.Debug("notification sent successfully")
	return nil
}

// Validate checks if the Apprise server is reachable.
func (a *AppriseClient) Validate(ctx context.Context) error {
	// Try to reach the Apprise server
	// Apprise API has various endpoints, try the details endpoint
	detailsURL := fmt.Sprintf("%s/details/%s", a.url, a.key)

	if err := a.httpClient.CheckConnectivity(ctx, detailsURL); err != nil {
		// Try the root URL as fallback
		if err2 := a.httpClient.CheckConnectivity(ctx, a.url); err2 != nil {
			return fmt.Errorf("apprise server not reachable at %s: %w", a.url, err)
		}
	}

	return nil
}

// mapLevel maps domain notification level to Apprise type.
func (a *AppriseClient) mapLevel(level domain.NotificationLevel) string {
	switch level {
	case domain.NotificationLevelInfo:
		return "info"
	case domain.NotificationLevelWarning:
		return "warning"
	case domain.NotificationLevelError:
		return "failure"
	default:
		return "info"
	}
}

// Ensure AppriseClient implements domain.Notifier.
var _ domain.Notifier = (*AppriseClient)(nil)
