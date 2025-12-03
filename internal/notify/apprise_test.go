package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sharkusmanch/ludusavi-runner/internal/domain"
)

func TestAppriseClient_Notify_Success(t *testing.T) {
	var receivedBody appriseRequest
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewAppriseClient(server.URL, "test-key")

	notification := domain.NewNotification("Test Title", "Test Body", domain.NotificationLevelError)
	err := client.Notify(context.Background(), notification)

	require.NoError(t, err)
	assert.Equal(t, "/notify/test-key", receivedPath)
	assert.Equal(t, "Test Title", receivedBody.Title)
	assert.Equal(t, "Test Body", receivedBody.Body)
	assert.Equal(t, "failure", receivedBody.Type)
}

func TestAppriseClient_Notify_TruncatesLongBody(t *testing.T) {
	var receivedBody appriseRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewAppriseClient(server.URL, "test-key")

	// Create a body longer than maxBodyLength
	longBody := strings.Repeat("a", 1500)
	notification := domain.NewNotification("Title", longBody, domain.NotificationLevelInfo)

	err := client.Notify(context.Background(), notification)

	require.NoError(t, err)
	assert.LessOrEqual(t, len(receivedBody.Body), maxBodyLength)
	assert.True(t, strings.HasSuffix(receivedBody.Body, "..."))
}

func TestAppriseClient_Notify_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client := NewAppriseClient(server.URL, "test-key")
	notification := domain.NewNotification("Title", "Body", domain.NotificationLevelError)

	err := client.Notify(context.Background(), notification)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestAppriseClient_Validate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewAppriseClient(server.URL, "test-key")
	err := client.Validate(context.Background())

	assert.NoError(t, err)
}

func TestAppriseClient_Validate_Failure(t *testing.T) {
	client := NewAppriseClient("http://localhost:1", "test-key")
	err := client.Validate(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not reachable")
}

func TestAppriseClient_MapLevel(t *testing.T) {
	client := NewAppriseClient("http://localhost", "key")

	tests := []struct {
		level    domain.NotificationLevel
		expected string
	}{
		{domain.NotificationLevelInfo, "info"},
		{domain.NotificationLevelWarning, "warning"},
		{domain.NotificationLevelError, "failure"},
		{domain.NotificationLevel("unknown"), "info"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			result := client.mapLevel(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}
