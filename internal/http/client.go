// Package http provides an HTTP client with retry logic and common utilities.
package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"time"
)

// RetryConfig configures retry behavior for the HTTP client.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (including the first).
	MaxAttempts int

	// InitialDelay is the delay before the first retry.
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration
}

// DefaultRetryConfig returns sensible default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 5 * time.Second,
		MaxDelay:     30 * time.Second,
	}
}

// Client is an HTTP client with retry logic.
type Client struct {
	httpClient *http.Client
	retry      RetryConfig
	logger     *slog.Logger
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithRetryConfig sets the retry configuration.
func WithRetryConfig(cfg RetryConfig) ClientOption {
	return func(c *Client) {
		c.retry = cfg
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// NewClient creates a new HTTP client with retry capabilities.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		retry:  DefaultRetryConfig(),
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Response wraps an HTTP response with convenience methods.
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Do performs an HTTP request with retry logic.
func (c *Client) Do(ctx context.Context, req *http.Request) (*Response, error) {
	var lastErr error
	var bodyBytes []byte

	// Read body for potential retries
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		_ = req.Body.Close()
	}

	for attempt := 1; attempt <= c.retry.MaxAttempts; attempt++ {
		// Reset body for each attempt
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		// Create a new request with context for each attempt
		attemptReq := req.Clone(ctx)

		c.logger.Debug("HTTP request attempt",
			"method", req.Method,
			"url", req.URL.String(),
			"attempt", attempt,
			"max_attempts", c.retry.MaxAttempts,
		)

		resp, err := c.httpClient.Do(attemptReq)
		if err != nil {
			lastErr = err
			c.logger.Warn("HTTP request failed",
				"method", req.Method,
				"url", req.URL.String(),
				"attempt", attempt,
				"error", err,
			)

			if attempt < c.retry.MaxAttempts {
				delay := c.calculateDelay(attempt)
				c.logger.Debug("Retrying after delay", "delay", delay)

				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(delay):
					continue
				}
			}
			continue
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		// Check for retryable status codes
		if c.shouldRetry(resp.StatusCode) && attempt < c.retry.MaxAttempts {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			c.logger.Warn("HTTP request returned retryable status",
				"status", resp.StatusCode,
				"attempt", attempt,
			)

			delay := c.calculateDelay(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		return &Response{
			StatusCode: resp.StatusCode,
			Body:       body,
			Headers:    resp.Header,
		}, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.retry.MaxAttempts, lastErr)
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, url string) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return c.Do(ctx, req)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, url string, contentType string, body []byte) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(ctx, req)
}

// calculateDelay calculates the delay for a given attempt using exponential backoff.
func (c *Client) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: initialDelay * 2^(attempt-1)
	delay := float64(c.retry.InitialDelay) * math.Pow(2, float64(attempt-1))

	if delay > float64(c.retry.MaxDelay) {
		return c.retry.MaxDelay
	}

	return time.Duration(delay)
}

// shouldRetry returns true if the status code indicates a retryable error.
func (c *Client) shouldRetry(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// CheckConnectivity performs a simple connectivity check to the given URL.
func (c *Client) CheckConnectivity(ctx context.Context, url string) error {
	// Use a shorter timeout for connectivity checks
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connectivity check failed: %w", err)
	}
	_ = resp.Body.Close()

	// Accept any 2xx or common endpoint responses
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("connectivity check returned status %d", resp.StatusCode)
}
