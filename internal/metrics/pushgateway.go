// Package metrics provides implementations for pushing metrics to remote endpoints.
package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"

	"github.com/sharkusmanch/ludusavi-runner/internal/domain"
	"github.com/sharkusmanch/ludusavi-runner/internal/http"
	"github.com/sharkusmanch/ludusavi-runner/pkg/version"
)

const (
	metricsJobName = "ludusavi"
	contentType    = "text/plain; charset=utf-8"
)

// PushgatewayClient pushes metrics to a Prometheus Pushgateway.
type PushgatewayClient struct {
	url        string
	httpClient *http.Client
	logger     *slog.Logger
}

// PushgatewayOption configures a PushgatewayClient.
type PushgatewayOption func(*PushgatewayClient)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) PushgatewayOption {
	return func(p *PushgatewayClient) {
		p.httpClient = client
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) PushgatewayOption {
	return func(p *PushgatewayClient) {
		p.logger = logger
	}
}

// NewPushgatewayClient creates a new PushgatewayClient.
func NewPushgatewayClient(url string, opts ...PushgatewayOption) *PushgatewayClient {
	p := &PushgatewayClient{
		url:        strings.TrimSuffix(url, "/"),
		httpClient: http.NewClient(),
		logger:     slog.Default(),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Push sends metrics to the Pushgateway.
func (p *PushgatewayClient) Push(ctx context.Context, metrics *domain.Metrics) error {
	body := p.buildMetrics(metrics)

	pushURL := fmt.Sprintf("%s/metrics/job/%s/instance/%s", p.url, metricsJobName, metrics.Hostname)

	p.logger.Debug("pushing metrics to pushgateway",
		"url", pushURL,
		"metrics_count", len(metrics.Results),
	)

	resp, err := p.httpClient.Post(ctx, pushURL, contentType, []byte(body))
	if err != nil {
		return fmt.Errorf("failed to push metrics: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pushgateway returned status %d: %s", resp.StatusCode, string(resp.Body))
	}

	p.logger.Debug("metrics pushed successfully")
	return nil
}

// Validate checks if the Pushgateway is reachable.
func (p *PushgatewayClient) Validate(ctx context.Context) error {
	// Pushgateway typically has a /-/ready endpoint
	readyURL := fmt.Sprintf("%s/-/ready", p.url)

	if err := p.httpClient.CheckConnectivity(ctx, readyURL); err != nil {
		// Try the root URL as fallback
		if err2 := p.httpClient.CheckConnectivity(ctx, p.url); err2 != nil {
			return fmt.Errorf("pushgateway not reachable at %s: %w", p.url, err)
		}
	}

	return nil
}

// buildMetrics constructs the Prometheus text format metrics.
func (p *PushgatewayClient) buildMetrics(m *domain.Metrics) string {
	var b strings.Builder

	// Service up metric
	b.WriteString("# HELP ludusavi_runner_up Service is running\n")
	b.WriteString("# TYPE ludusavi_runner_up gauge\n")
	if m.ServiceUp {
		b.WriteString("ludusavi_runner_up 1\n")
	} else {
		b.WriteString("ludusavi_runner_up 0\n")
	}
	b.WriteString("\n")

	// Info metric
	versionInfo := version.Get()
	b.WriteString("# HELP ludusavi_runner_info Build information\n")
	b.WriteString("# TYPE ludusavi_runner_info gauge\n")
	b.WriteString(fmt.Sprintf("ludusavi_runner_info{version=%q,go_version=%q} 1\n",
		versionInfo.Version, runtime.Version()))
	b.WriteString("\n")

	// Result metrics
	for _, result := range m.Results {
		p.writeResultMetrics(&b, result)
	}

	return b.String()
}

// writeResultMetrics writes metrics for a single backup result.
func (p *PushgatewayClient) writeResultMetrics(b *strings.Builder, r *domain.BackupResult) {
	op := r.Operation.String()

	// Last run timestamp
	b.WriteString("# HELP ludusavi_last_run_timestamp_seconds Unix timestamp of last run\n")
	b.WriteString("# TYPE ludusavi_last_run_timestamp_seconds gauge\n")
	b.WriteString(fmt.Sprintf("ludusavi_last_run_timestamp_seconds{operation=%q} %d\n",
		op, r.EndTime.Unix()))
	b.WriteString("\n")

	// Last run success
	b.WriteString("# HELP ludusavi_last_run_success Whether the last run succeeded\n")
	b.WriteString("# TYPE ludusavi_last_run_success gauge\n")
	success := 0
	if r.Success {
		success = 1
	}
	b.WriteString(fmt.Sprintf("ludusavi_last_run_success{operation=%q} %d\n", op, success))
	b.WriteString("\n")

	// Last run duration
	b.WriteString("# HELP ludusavi_last_run_duration_seconds Duration of last run\n")
	b.WriteString("# TYPE ludusavi_last_run_duration_seconds gauge\n")
	b.WriteString(fmt.Sprintf("ludusavi_last_run_duration_seconds{operation=%q} %.3f\n",
		op, r.Duration.Seconds()))
	b.WriteString("\n")

	// Games total
	b.WriteString("# HELP ludusavi_games_total Total games detected\n")
	b.WriteString("# TYPE ludusavi_games_total gauge\n")
	b.WriteString(fmt.Sprintf("ludusavi_games_total{operation=%q} %d\n", op, r.Stats.TotalGames))
	b.WriteString("\n")

	// Games processed
	b.WriteString("# HELP ludusavi_games_processed Games processed in last run\n")
	b.WriteString("# TYPE ludusavi_games_processed gauge\n")
	b.WriteString(fmt.Sprintf("ludusavi_games_processed{operation=%q} %d\n", op, r.Stats.ProcessedGames))
	b.WriteString("\n")

	// Bytes total
	b.WriteString("# HELP ludusavi_bytes_total Total bytes across all saves\n")
	b.WriteString("# TYPE ludusavi_bytes_total gauge\n")
	b.WriteString(fmt.Sprintf("ludusavi_bytes_total{operation=%q} %d\n", op, r.Stats.TotalBytes))
	b.WriteString("\n")

	// Bytes processed
	b.WriteString("# HELP ludusavi_bytes_processed Bytes processed in last run\n")
	b.WriteString("# TYPE ludusavi_bytes_processed gauge\n")
	b.WriteString(fmt.Sprintf("ludusavi_bytes_processed{operation=%q} %d\n", op, r.Stats.ProcessedBytes))
	b.WriteString("\n")

	// Games new
	b.WriteString("# HELP ludusavi_games_new New games backed up\n")
	b.WriteString("# TYPE ludusavi_games_new gauge\n")
	b.WriteString(fmt.Sprintf("ludusavi_games_new{operation=%q} %d\n", op, r.Stats.NewGames))
	b.WriteString("\n")

	// Games changed
	b.WriteString("# HELP ludusavi_games_changed Games with changes\n")
	b.WriteString("# TYPE ludusavi_games_changed gauge\n")
	b.WriteString(fmt.Sprintf("ludusavi_games_changed{operation=%q} %d\n", op, r.Stats.ChangedGames))
	b.WriteString("\n")
}

// Ensure PushgatewayClient implements domain.MetricsPusher.
var _ domain.MetricsPusher = (*PushgatewayClient)(nil)
