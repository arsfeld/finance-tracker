package scheduler

import (
	"context"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

// Scheduler wraps robfig/cron with mutex-based job exclusion.
type Scheduler struct {
	cron    *cron.Cron
	mu      sync.Mutex
	running string // name of the currently running job, empty if idle
}

// New creates a new Scheduler with SkipIfStillRunning semantics.
func New() *Scheduler {
	c := cron.New(
		cron.WithSeconds(),
		cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)),
	)
	return &Scheduler{cron: c}
}

// AddFunc adds a scheduled function with a cron expression.
func (s *Scheduler) AddFunc(spec string, cmd func()) error {
	_, err := s.cron.AddFunc(spec, cmd)
	return err
}

// Start begins the cron scheduler.
func (s *Scheduler) Start() {
	s.cron.Start()
	log.Info().Msg("Scheduler started")
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop(ctx context.Context) {
	stopCtx := s.cron.Stop()
	select {
	case <-stopCtx.Done():
		log.Info().Msg("Scheduler stopped, all jobs finished")
	case <-ctx.Done():
		log.Warn().Msg("Scheduler stop timed out")
	}
}

// TryAcquire attempts to acquire the job mutex for an operation.
// Returns true if acquired (caller must call Release when done).
// Returns false if another operation is already running.
func (s *Scheduler) TryAcquire(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running != "" {
		return false
	}
	s.running = name
	return true
}

// Release releases the job mutex.
func (s *Scheduler) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = ""
}

// CurrentJob returns the name of the currently running job, or empty string if idle.
func (s *Scheduler) CurrentJob() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
