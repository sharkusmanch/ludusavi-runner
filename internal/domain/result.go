// Package domain defines core business types and interfaces.
package domain

import "time"

// OperationType represents the type of backup operation.
type OperationType string

const (
	// OperationBackup represents a local backup operation.
	OperationBackup OperationType = "backup"
	// OperationCloudUpload represents a cloud upload operation.
	OperationCloudUpload OperationType = "cloud_upload"
)

// String returns the string representation of the operation type.
func (o OperationType) String() string {
	return string(o)
}

// BackupStats contains statistics from a backup or upload operation.
type BackupStats struct {
	TotalGames     int   `json:"total_games"`
	ProcessedGames int   `json:"processed_games"`
	TotalBytes     int64 `json:"total_bytes"`
	ProcessedBytes int64 `json:"processed_bytes"`
	NewGames       int   `json:"new_games"`
	ChangedGames   int   `json:"changed_games"`
	SameGames      int   `json:"same_games"`
}

// BackupResult contains the result of a backup operation.
type BackupResult struct {
	Operation OperationType `json:"operation"`
	Success   bool          `json:"success"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Stats     BackupStats   `json:"stats"`
	Error     string        `json:"error,omitempty"`
}

// NewBackupResult creates a new BackupResult with the given operation type.
func NewBackupResult(op OperationType) *BackupResult {
	return &BackupResult{
		Operation: op,
		StartTime: time.Now(),
	}
}

// Complete marks the result as complete.
func (r *BackupResult) Complete(success bool, err error) {
	r.EndTime = time.Now()
	r.Duration = r.EndTime.Sub(r.StartTime)
	r.Success = success
	if err != nil {
		r.Error = err.Error()
	}
}

// RunResult contains the results of a complete backup run (all operations).
type RunResult struct {
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	Duration    time.Duration `json:"duration"`
	Success     bool          `json:"success"`
	DryRun      bool          `json:"dry_run"`
	Backup      *BackupResult `json:"backup,omitempty"`
	CloudUpload *BackupResult `json:"cloud_upload,omitempty"`
	Errors      []string      `json:"errors,omitempty"`
}

// NewRunResult creates a new RunResult.
func NewRunResult(dryRun bool) *RunResult {
	return &RunResult{
		StartTime: time.Now(),
		DryRun:    dryRun,
		Errors:    make([]string, 0),
	}
}

// Complete marks the run as complete.
func (r *RunResult) Complete() {
	r.EndTime = time.Now()
	r.Duration = r.EndTime.Sub(r.StartTime)

	// Success if both operations succeeded (or were not run)
	r.Success = true
	if r.CloudUpload != nil && !r.CloudUpload.Success {
		r.Success = false
	}
	if r.Backup != nil && !r.Backup.Success {
		r.Success = false
	}
}

// AddError adds an error to the run result.
func (r *RunResult) AddError(err error) {
	if err != nil {
		r.Errors = append(r.Errors, err.Error())
	}
}
