// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package scheduler provides built-in task scheduling per AI.md PART 19
// All projects MUST have a built-in scheduler that is ALWAYS RUNNING
// NEVER use external schedulers (cron, systemd timers, Task Scheduler, etc.)
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TaskStatus represents the status of a scheduled task
type TaskStatus string

const (
	// StatusPending means the task is waiting to run
	StatusPending TaskStatus = "pending"
	// StatusRunning means the task is currently executing
	StatusRunning TaskStatus = "running"
	// StatusComplete means the task completed successfully
	StatusComplete TaskStatus = "complete"
	// StatusFailed means the task failed
	StatusFailed TaskStatus = "failed"
	// StatusSkipped means the task was skipped
	StatusSkipped TaskStatus = "skipped"
)

// Task represents a scheduled task
type Task struct {
	ID          string
	Name        string
	Description string
	Schedule    string
	Enabled     bool
	Skippable   bool
	RetryOnFail bool
	RetryDelay  time.Duration
	Handler     func(ctx context.Context) error
	LastRun     time.Time
	NextRun     time.Time
	LastStatus  TaskStatus
	LastError   string
	RunCount    int64
	FailCount   int64
	cronExpr    *CronExpr
	mu          sync.RWMutex
}

// Config holds scheduler configuration
type Config struct {
	// Timezone for scheduled tasks (default: America/New_York)
	Timezone string
	// CatchUpWindow is the duration within which missed tasks are run
	CatchUpWindow time.Duration
}

// DefaultConfig returns the default scheduler configuration
func DefaultConfig() *Config {
	return &Config{
		Timezone:      "America/New_York",
		CatchUpWindow: time.Hour,
	}
}

// Scheduler manages scheduled tasks
type Scheduler struct {
	config   *Config
	tasks    map[string]*Task
	running  bool
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.RWMutex
	wg       sync.WaitGroup
	location *time.Location
}

// New creates a new scheduler
func New(cfg *Config) *Scheduler {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		loc = time.Local
	}

	return &Scheduler{
		config:   cfg,
		tasks:    make(map[string]*Task),
		location: loc,
	}
}

// AddTask adds a task to the scheduler
func (s *Scheduler) AddTask(task *Task) error {
	if task.ID == "" {
		return fmt.Errorf("task ID is required")
	}
	if task.Handler == nil {
		return fmt.Errorf("task handler is required")
	}

	// Parse cron expression if provided
	if task.Schedule != "" {
		cronExpr, err := ParseCron(task.Schedule)
		if err != nil {
			return fmt.Errorf("invalid schedule for task %s: %w", task.ID, err)
		}
		task.cronExpr = cronExpr
		// Calculate initial next run
		task.NextRun = cronExpr.Next(time.Now().In(s.location))
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[task.ID] = task
	return nil
}

// RemoveTask removes a task from the scheduler
func (s *Scheduler) RemoveTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return fmt.Errorf("task not found: %s", id)
	}

	delete(s.tasks, id)
	return nil
}

// GetTask returns a task by ID
func (s *Scheduler) GetTask(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]
	return task, ok
}

// ListTasks returns all tasks
func (s *Scheduler) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.running = true
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.mu.Unlock()

	// Run catch-up for missed tasks
	s.runCatchUp()

	// Start the main scheduler loop
	s.wg.Add(1)
	go s.run()

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler not running")
	}
	s.running = false
	s.cancel()
	s.mu.Unlock()

	s.wg.Wait()
	return nil
}

// IsRunning returns true if the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case now := <-ticker.C:
			s.checkAndRunTasks(now)
		}
	}
}

// checkAndRunTasks checks all tasks and runs those that are due
func (s *Scheduler) checkAndRunTasks(now time.Time) {
	s.mu.RLock()
	tasks := make([]*Task, 0)
	for _, task := range s.tasks {
		if task.Enabled && !task.NextRun.IsZero() && now.After(task.NextRun) {
			tasks = append(tasks, task)
		}
	}
	s.mu.RUnlock()

	for _, task := range tasks {
		s.runTask(task)
	}
}

// runTask executes a single task
func (s *Scheduler) runTask(task *Task) {
	task.mu.Lock()
	if task.LastStatus == StatusRunning {
		task.mu.Unlock()
		return
	}
	task.LastStatus = StatusRunning
	task.mu.Unlock()

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	err := task.Handler(ctx)

	task.mu.Lock()
	task.LastRun = time.Now()
	task.RunCount++
	if err != nil {
		task.LastStatus = StatusFailed
		task.LastError = err.Error()
		task.FailCount++
	} else {
		task.LastStatus = StatusComplete
		task.LastError = ""
	}
	// Calculate next run using cron expression
	if task.cronExpr != nil {
		task.NextRun = task.cronExpr.Next(time.Now().In(s.location))
	} else {
		task.NextRun = time.Time{}
	}
	task.mu.Unlock()
}

// runCatchUp runs tasks that were missed while the scheduler was stopped
func (s *Scheduler) runCatchUp() {
	now := time.Now()
	cutoff := now.Add(-s.config.CatchUpWindow)

	s.mu.RLock()
	tasks := make([]*Task, 0)
	for _, task := range s.tasks {
		if task.Enabled && !task.LastRun.IsZero() && task.LastRun.Before(cutoff) {
			if !task.Skippable {
				tasks = append(tasks, task)
			}
		}
	}
	s.mu.RUnlock()

	for _, task := range tasks {
		s.runTask(task)
	}
}

// RunNow immediately runs a task regardless of schedule
func (s *Scheduler) RunNow(id string) error {
	task, ok := s.GetTask(id)
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}

	go s.runTask(task)
	return nil
}

// GetStatus returns the scheduler status
func (s *Scheduler) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	taskStatuses := make([]map[string]interface{}, 0, len(s.tasks))
	for _, task := range s.tasks {
		task.mu.RLock()
		taskStatuses = append(taskStatuses, map[string]interface{}{
			"id":          task.ID,
			"name":        task.Name,
			"enabled":     task.Enabled,
			"last_run":    task.LastRun,
			"next_run":    task.NextRun,
			"last_status": task.LastStatus,
			"run_count":   task.RunCount,
			"fail_count":  task.FailCount,
		})
		task.mu.RUnlock()
	}

	return map[string]interface{}{
		"running":  s.running,
		"timezone": s.config.Timezone,
		"tasks":    taskStatuses,
	}
}
