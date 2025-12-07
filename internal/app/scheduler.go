package app

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/sharkusmanch/ludusavi-runner/internal/domain"
)

// Scheduler manages periodic execution of backup runs.
type Scheduler struct {
	runner          *Runner
	interval        time.Duration
	backupOnStartup bool
	logger          *slog.Logger

	mu        sync.Mutex
	running   bool
	stopCh    chan struct{}
	stoppedCh chan struct{}
}

// SchedulerOption configures a Scheduler.
type SchedulerOption func(*Scheduler)

// WithInterval sets the backup interval.
func WithInterval(d time.Duration) SchedulerOption {
	return func(s *Scheduler) {
		s.interval = d
	}
}

// WithBackupOnStartup sets whether to run a backup immediately on start.
func WithBackupOnStartup(b bool) SchedulerOption {
	return func(s *Scheduler) {
		s.backupOnStartup = b
	}
}

// WithSchedulerLogger sets the logger.
func WithSchedulerLogger(l *slog.Logger) SchedulerOption {
	return func(s *Scheduler) {
		s.logger = l
	}
}

// NewScheduler creates a new Scheduler.
func NewScheduler(runner *Runner, opts ...SchedulerOption) *Scheduler {
	s := &Scheduler{
		runner:          runner,
		interval:        20 * time.Minute,
		backupOnStartup: true,
		logger:          slog.Default(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Start begins the scheduler loop. It runs until Stop is called or the context is cancelled.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.stoppedCh = make(chan struct{})
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		close(s.stoppedCh)
		s.mu.Unlock()
	}()

	s.logger.Info("scheduler started",
		"interval", s.interval,
		"backup_on_startup", s.backupOnStartup,
	)

	// Run backup on startup if configured
	if s.backupOnStartup {
		s.logger.Debug("running backup on startup")
		s.runBackup(ctx)
	}

	// Schedule periodic backups
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopping due to context cancellation")
			s.runFinalBackup()
			return ctx.Err()

		case <-s.stopCh:
			s.logger.Info("scheduler stopping due to stop signal")
			s.runFinalBackup()
			return nil

		case <-ticker.C:
			s.logger.Debug("interval triggered, running backup")
			s.runBackup(ctx)
		}
	}
}

// runBackup runs a backup with a separate context that allows graceful completion.
// If shutdown is requested during a backup, the backup gets a 2 minute grace period.
func (s *Scheduler) runBackup(ctx context.Context) {
	// Check if shutdown was already requested before starting
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Create a backup context that allows graceful completion
	backupCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	// Monitor for shutdown and give grace period
	go func() {
		select {
		case <-done:
			// Backup completed normally
		case <-ctx.Done():
			// Shutdown requested - give grace period then cancel
			s.logger.Info("shutdown requested, allowing backup to complete (2m grace period)")
			select {
			case <-done:
			case <-time.After(2 * time.Minute):
				s.logger.Warn("backup grace period expired, cancelling")
				cancel()
			}
		}
	}()

	if _, err := s.runner.Run(backupCtx); err != nil {
		s.logger.Error("backup failed", "error", err)
	}
	close(done)
	cancel()
}

// Stop signals the scheduler to stop.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	close(s.stopCh)
	stoppedCh := s.stoppedCh
	s.mu.Unlock()

	// Wait for the scheduler to actually stop
	<-stoppedCh
}

// IsRunning returns true if the scheduler is currently running.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// runFinalBackup pushes a final metrics update before stopping.
func (s *Scheduler) runFinalBackup() {
	s.logger.Debug("pushing final metrics before shutdown")

	// Create a context with timeout for the final push
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Push a final "service down" metric
	if s.runner.metricsPusher != nil {
		metrics := domain.NewMetrics(s.runner.hostname)
		metrics.ServiceUp = false
		if err := s.runner.metricsPusher.Push(ctx, metrics); err != nil {
			s.logger.Warn("failed to push final metrics", "error", err)
		}
	}
}
