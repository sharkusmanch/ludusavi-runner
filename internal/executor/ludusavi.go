// Package executor provides implementations of the Executor interface.
package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sharkusmanch/ludusavi-runner/internal/domain"
)

// LudusaviOutput represents the JSON output from ludusavi --api commands.
type LudusaviOutput struct {
	Overall LudusaviOverall `json:"overall"`
	Errors  LudusaviErrors  `json:"errors,omitempty"`
}

// LudusaviOverall contains the overall statistics from ludusavi.
type LudusaviOverall struct {
	TotalGames     int                  `json:"totalGames"`
	TotalBytes     int64                `json:"totalBytes"`
	ProcessedGames int                  `json:"processedGames"`
	ProcessedBytes int64                `json:"processedBytes"`
	ChangedGames   LudusaviChangedGames `json:"changedGames"`
}

// LudusaviChangedGames contains the breakdown of changed games.
type LudusaviChangedGames struct {
	New       int `json:"new"`
	Different int `json:"different"`
	Same      int `json:"same"`
}

// LudusaviErrors contains error information from ludusavi.
type LudusaviErrors struct {
	SomeGamesFailed bool `json:"someGamesFailed"`
}

// LudusaviExecutor implements Executor using the ludusavi CLI.
type LudusaviExecutor struct {
	binaryPath string
	env        map[string]string
	logger     *slog.Logger
}

// LudusaviOption configures a LudusaviExecutor.
type LudusaviOption func(*LudusaviExecutor)

// WithBinaryPath sets the path to the ludusavi binary.
func WithBinaryPath(path string) LudusaviOption {
	return func(e *LudusaviExecutor) {
		e.binaryPath = path
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) LudusaviOption {
	return func(e *LudusaviExecutor) {
		e.logger = logger
	}
}

// WithEnv sets environment variables to pass to ludusavi.
func WithEnv(env map[string]string) LudusaviOption {
	return func(e *LudusaviExecutor) {
		e.env = env
	}
}

// NewLudusaviExecutor creates a new LudusaviExecutor.
func NewLudusaviExecutor(opts ...LudusaviOption) *LudusaviExecutor {
	e := &LudusaviExecutor{
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Backup runs a local backup operation.
func (e *LudusaviExecutor) Backup(ctx context.Context, opts domain.BackupOptions) (*domain.BackupResult, error) {
	result := domain.NewBackupResult(domain.OperationBackup)

	args := []string{"backup", "--api"}
	if opts.Force {
		args = append(args, "--force")
	}

	output, err := e.run(ctx, args...)
	if err != nil {
		result.Complete(false, err)
		return result, nil
	}

	stats, err := e.parseOutput(output)
	if err != nil {
		result.Complete(false, fmt.Errorf("failed to parse output: %w", err))
		return result, nil
	}

	result.Stats = *stats
	result.Complete(true, nil)
	return result, nil
}

// CloudUpload runs a cloud upload operation.
func (e *LudusaviExecutor) CloudUpload(ctx context.Context, opts domain.UploadOptions) (*domain.BackupResult, error) {
	result := domain.NewBackupResult(domain.OperationCloudUpload)

	args := []string{"cloud", "upload", "--api"}
	if opts.Force {
		args = append(args, "--force")
	}

	output, err := e.run(ctx, args...)
	if err != nil {
		result.Complete(false, err)
		return result, nil
	}

	stats, err := e.parseOutput(output)
	if err != nil {
		result.Complete(false, fmt.Errorf("failed to parse output: %w", err))
		return result, nil
	}

	result.Stats = *stats
	result.Complete(true, nil)
	return result, nil
}

// Version returns the ludusavi version.
func (e *LudusaviExecutor) Version(ctx context.Context) (string, error) {
	output, err := e.run(ctx, "--version")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// Validate checks if ludusavi is properly configured and available.
func (e *LudusaviExecutor) Validate(ctx context.Context) error {
	path, err := e.getBinaryPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("ludusavi binary not found at %s: %w", path, err)
	}

	// Try to get version to verify it works
	_, err = e.Version(ctx)
	if err != nil {
		return fmt.Errorf("ludusavi binary failed to execute: %w", err)
	}

	return nil
}

// run executes ludusavi with the given arguments.
func (e *LudusaviExecutor) run(ctx context.Context, args ...string) ([]byte, error) {
	path, err := e.getBinaryPath()
	if err != nil {
		return nil, err
	}

	e.logger.Debug("executing ludusavi", "path", path, "args", args)

	// #nosec G204 -- path is from config or auto-detected, not user input
	cmd := exec.CommandContext(ctx, path, args...)

	// Set environment variables if configured
	if len(e.env) > 0 {
		// Start with current environment and add/override with configured vars
		cmd.Env = os.Environ()
		for k, v := range e.env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Check if it's a context error
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Include stderr in error message
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return nil, fmt.Errorf("ludusavi failed: %s: %w", errMsg, err)
		}
		return nil, fmt.Errorf("ludusavi failed: %w", err)
	}

	return stdout.Bytes(), nil
}

// parseOutput parses the JSON output from ludusavi.
func (e *LudusaviExecutor) parseOutput(output []byte) (*domain.BackupStats, error) {
	// Handle empty output (e.g., cloud upload with nothing to sync)
	if len(bytes.TrimSpace(output)) == 0 {
		return &domain.BackupStats{}, nil
	}

	var ludusaviOut LudusaviOutput
	if err := json.Unmarshal(output, &ludusaviOut); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &domain.BackupStats{
		TotalGames:     ludusaviOut.Overall.TotalGames,
		ProcessedGames: ludusaviOut.Overall.ProcessedGames,
		TotalBytes:     ludusaviOut.Overall.TotalBytes,
		ProcessedBytes: ludusaviOut.Overall.ProcessedBytes,
		NewGames:       ludusaviOut.Overall.ChangedGames.New,
		ChangedGames:   ludusaviOut.Overall.ChangedGames.Different,
		SameGames:      ludusaviOut.Overall.ChangedGames.Same,
	}, nil
}

// getBinaryPath returns the path to the ludusavi binary.
func (e *LudusaviExecutor) getBinaryPath() (string, error) {
	// Use configured path if set
	if e.binaryPath != "" {
		return e.binaryPath, nil
	}

	// Try to find in PATH
	path, err := exec.LookPath("ludusavi")
	if err == nil {
		return path, nil
	}

	// Try common locations based on OS
	candidates := e.getCommonPaths()
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("ludusavi not found in PATH or common locations")
}

// getCommonPaths returns common installation paths for ludusavi.
func (e *LudusaviExecutor) getCommonPaths() []string {
	switch runtime.GOOS {
	case "windows":
		home := os.Getenv("USERPROFILE")
		return []string{
			filepath.Join(home, "scoop", "shims", "ludusavi.exe"),
			filepath.Join(home, "scoop", "apps", "ludusavi", "current", "ludusavi.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "ludusavi", "ludusavi.exe"),
			"C:\\Program Files\\ludusavi\\ludusavi.exe",
		}
	case "darwin":
		home, _ := os.UserHomeDir()
		return []string{
			"/usr/local/bin/ludusavi",
			"/opt/homebrew/bin/ludusavi",
			filepath.Join(home, ".local", "bin", "ludusavi"),
		}
	default: // Linux and others
		home, _ := os.UserHomeDir()
		return []string{
			"/usr/bin/ludusavi",
			"/usr/local/bin/ludusavi",
			filepath.Join(home, ".local", "bin", "ludusavi"),
			filepath.Join(home, ".cargo", "bin", "ludusavi"),
		}
	}
}
