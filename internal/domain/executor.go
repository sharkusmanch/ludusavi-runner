package domain

import "context"

// BackupOptions contains options for a backup operation.
type BackupOptions struct {
	// Force skips confirmation prompts.
	Force bool
}

// UploadOptions contains options for a cloud upload operation.
type UploadOptions struct {
	// Force skips confirmation prompts.
	Force bool
}

// Executor defines the interface for running backup operations.
// This abstraction allows for different implementations (real ludusavi, mock, etc.).
type Executor interface {
	// Backup runs a local backup operation and returns the result.
	Backup(ctx context.Context, opts BackupOptions) (*BackupResult, error)

	// CloudUpload runs a cloud upload operation and returns the result.
	CloudUpload(ctx context.Context, opts UploadOptions) (*BackupResult, error)

	// Version returns the ludusavi version string.
	Version(ctx context.Context) (string, error)

	// Validate checks if the executor is properly configured.
	Validate(ctx context.Context) error
}
