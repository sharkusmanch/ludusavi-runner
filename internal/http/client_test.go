package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Get(context.Background(), server.URL)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "success", string(resp.Body))
}

func TestClient_Post_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Post(context.Background(), server.URL, "application/json", []byte(`{"test":true}`))

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_Retry_Success(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)
		if attempt < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))
	defer server.Close()

	client := NewClient(WithRetryConfig(RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}))

	resp, err := client.Get(context.Background(), server.URL)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

func TestClient_Retry_Exhausted(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewClient(WithRetryConfig(RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}))

	resp, err := client.Get(context.Background(), server.URL)

	// Should return the last response, not an error
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

func TestClient_Retry_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewClient(WithRetryConfig(RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 1 * time.Second,
		MaxDelay:     5 * time.Second,
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Get(ctx, server.URL)

	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestClient_NoRetry_ClientError(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(WithRetryConfig(RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}))

	resp, err := client.Get(context.Background(), server.URL)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	// Should not retry on 4xx errors
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
}

func TestClient_CheckConnectivity_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	err := client.CheckConnectivity(context.Background(), server.URL)

	assert.NoError(t, err)
}

func TestClient_CheckConnectivity_Failure(t *testing.T) {
	client := NewClient()
	err := client.CheckConnectivity(context.Background(), "http://localhost:1")

	assert.Error(t, err)
}

func TestCalculateDelay(t *testing.T) {
	client := NewClient(WithRetryConfig(RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
	}))

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 1 * time.Second},  // 1 * 2^0 = 1
		{2, 2 * time.Second},  // 1 * 2^1 = 2
		{3, 4 * time.Second},  // 1 * 2^2 = 4
		{4, 8 * time.Second},  // 1 * 2^3 = 8
		{5, 10 * time.Second}, // 1 * 2^4 = 16, capped at 10
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			delay := client.calculateDelay(tt.attempt)
			assert.Equal(t, tt.expected, delay)
		})
	}
}

func TestShouldRetry(t *testing.T) {
	client := NewClient()

	retryable := []int{
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	}

	notRetryable := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
	}

	for _, code := range retryable {
		assert.True(t, client.shouldRetry(code), "expected %d to be retryable", code)
	}

	for _, code := range notRetryable {
		assert.False(t, client.shouldRetry(code), "expected %d to not be retryable", code)
	}
}
