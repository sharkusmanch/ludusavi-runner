package executor

import (
	"context"

	"github.com/sharkusmanch/ludusavi-runner/internal/domain"
)

// MockExecutor is a mock implementation of domain.Executor for testing.
type MockExecutor struct {
	BackupFunc      func(ctx context.Context, opts domain.BackupOptions) (*domain.BackupResult, error)
	CloudUploadFunc func(ctx context.Context, opts domain.UploadOptions) (*domain.BackupResult, error)
	VersionFunc     func(ctx context.Context) (string, error)
	ValidateFunc    func(ctx context.Context) error
}

// Backup calls the mock BackupFunc.
func (m *MockExecutor) Backup(ctx context.Context, opts domain.BackupOptions) (*domain.BackupResult, error) {
	if m.BackupFunc != nil {
		return m.BackupFunc(ctx, opts)
	}
	result := domain.NewBackupResult(domain.OperationBackup)
	result.Complete(true, nil)
	return result, nil
}

// CloudUpload calls the mock CloudUploadFunc.
func (m *MockExecutor) CloudUpload(ctx context.Context, opts domain.UploadOptions) (*domain.BackupResult, error) {
	if m.CloudUploadFunc != nil {
		return m.CloudUploadFunc(ctx, opts)
	}
	result := domain.NewBackupResult(domain.OperationCloudUpload)
	result.Complete(true, nil)
	return result, nil
}

// Version calls the mock VersionFunc.
func (m *MockExecutor) Version(ctx context.Context) (string, error) {
	if m.VersionFunc != nil {
		return m.VersionFunc(ctx)
	}
	return "mock-version", nil
}

// Validate calls the mock ValidateFunc.
func (m *MockExecutor) Validate(ctx context.Context) error {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(ctx)
	}
	return nil
}

// Ensure MockExecutor implements domain.Executor.
var _ domain.Executor = (*MockExecutor)(nil)
