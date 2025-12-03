package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sharkusmanch/ludusavi-runner/internal/domain"
)

func TestPushgatewayClient_Push_Success(t *testing.T) {
	var receivedBody string
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewPushgatewayClient(server.URL)

	metrics := domain.NewMetrics("test-host")
	metrics.ServiceUp = true

	result := domain.NewBackupResult(domain.OperationBackup)
	result.Stats = domain.BackupStats{
		TotalGames:     100,
		ProcessedGames: 95,
		TotalBytes:     1000000,
		ProcessedBytes: 950000,
		NewGames:       5,
		ChangedGames:   10,
	}
	result.Complete(true, nil)
	metrics.AddResult(result)

	err := client.Push(context.Background(), metrics)

	require.NoError(t, err)
	assert.Equal(t, "/metrics/job/ludusavi/instance/test-host", receivedPath)
	assert.Contains(t, receivedBody, "ludusavi_runner_up 1")
	assert.Contains(t, receivedBody, "ludusavi_games_total")
	assert.Contains(t, receivedBody, `operation="backup"`)
}

func TestPushgatewayClient_Push_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewPushgatewayClient(server.URL)
	metrics := domain.NewMetrics("test-host")

	err := client.Push(context.Background(), metrics)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestPushgatewayClient_Validate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewPushgatewayClient(server.URL)
	err := client.Validate(context.Background())

	assert.NoError(t, err)
}

func TestPushgatewayClient_Validate_Failure(t *testing.T) {
	client := NewPushgatewayClient("http://localhost:1")
	err := client.Validate(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not reachable")
}

func TestPushgatewayClient_BuildMetrics(t *testing.T) {
	client := NewPushgatewayClient("http://localhost:9091")

	metrics := domain.NewMetrics("test-host")
	metrics.ServiceUp = true
	metrics.Version = "1.0.0"

	backupResult := &domain.BackupResult{
		Operation: domain.OperationBackup,
		Success:   true,
		StartTime: time.Now().Add(-5 * time.Second),
		EndTime:   time.Now(),
		Duration:  5 * time.Second,
		Stats: domain.BackupStats{
			TotalGames:     178,
			ProcessedGames: 174,
			TotalBytes:     3353481924,
			ProcessedBytes: 3330568438,
			NewGames:       2,
			ChangedGames:   5,
			SameGames:      167,
		},
	}
	metrics.AddResult(backupResult)

	uploadResult := &domain.BackupResult{
		Operation: domain.OperationCloudUpload,
		Success:   true,
		StartTime: time.Now().Add(-10 * time.Second),
		EndTime:   time.Now().Add(-5 * time.Second),
		Duration:  5 * time.Second,
		Stats: domain.BackupStats{
			TotalGames:     50,
			ProcessedGames: 50,
		},
	}
	metrics.AddResult(uploadResult)

	body := client.buildMetrics(metrics)

	// Check for expected metrics
	assert.Contains(t, body, "ludusavi_runner_up 1")
	assert.Contains(t, body, "ludusavi_runner_info")
	assert.Contains(t, body, `operation="backup"`)
	assert.Contains(t, body, `operation="cloud_upload"`)
	assert.Contains(t, body, "ludusavi_last_run_success")
	assert.Contains(t, body, "ludusavi_last_run_duration_seconds")
	assert.Contains(t, body, "ludusavi_games_total")
	assert.Contains(t, body, "ludusavi_bytes_total")

	// Verify valid Prometheus format (no syntax errors)
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Each non-comment, non-empty line should have a metric name and value
		parts := strings.Fields(line)
		assert.GreaterOrEqual(t, len(parts), 2, "line should have metric and value: %s", line)
	}
}

func TestPushgatewayClient_BuildMetrics_ServiceDown(t *testing.T) {
	client := NewPushgatewayClient("http://localhost:9091")

	metrics := domain.NewMetrics("test-host")
	metrics.ServiceUp = false

	body := client.buildMetrics(metrics)

	assert.Contains(t, body, "ludusavi_runner_up 0")
}
