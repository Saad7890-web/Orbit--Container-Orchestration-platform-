package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Saad7890-web/orbit/internal/models"
)

type ScheduledJob struct {
	StackName  string
	Job        models.Job
	ConfigHash string
	Schedule   *CronSchedule
	NextRun    time.Time
	Running    bool
	Active     bool
}

type Scheduler struct {
	mu       sync.RWMutex
	jobs     map[string]*ScheduledJob
	runner   *JobRunner
	interval time.Duration
}

func New(runner *JobRunner) *Scheduler {
	return &Scheduler{
		jobs:     make(map[string]*ScheduledJob),
		runner:   runner,
		interval: 5 * time.Second,
	}
}

func (s *Scheduler) SetInterval(interval time.Duration) {
	if interval <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interval = interval
}

func (s *Scheduler) Register(stackName string, job models.Job, configHash string) error {
	if s == nil {
		return fmt.Errorf("scheduler is nil")
	}
	if job.Schedule == nil {
		return fmt.Errorf("job %q has no schedule", job.Name)
	}

	cr, err := ParseSchedule(*job.Schedule)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	next := cr.Next(now)
	if next.IsZero() {
		return fmt.Errorf("job %q has no next run within search horizon", job.Name)
	}

	key := jobKey(stackName, job.Name)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs[key] = &ScheduledJob{
		StackName:  stackName,
		Job:        job,
		ConfigHash: configHash,
		Schedule:   cr,
		NextRun:    next,
		Active:     true,
	}

	return nil
}

func (s *Scheduler) Unregister(stackName, jobName string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.jobs, jobKey(stackName, jobName))
}

func (s *Scheduler) Sync(stackName string, jobs []models.Job, configHash string) error {
	if s == nil {
		return fmt.Errorf("scheduler is nil")
	}

	desired := make(map[string]struct{}, len(jobs))
	for _, job := range jobs {
		desired[jobKey(stackName, job.Name)] = struct{}{}
		if job.Schedule == nil {
			continue
		}
		if err := s.Register(stackName, job, configHash); err != nil {
			return err
		}
	}

	s.mu.Lock()
	for key := range s.jobs {
		if _, ok := desired[key]; !ok {
			delete(s.jobs, key)
		}
	}
	s.mu.Unlock()

	return nil
}

func (s *Scheduler) Start(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("scheduler is nil")
	}
	ticker := time.NewTicker(s.getInterval())
	defer ticker.Stop()

	for {
		if err := s.dispatchDue(ctx); err != nil {
			// Keep the scheduler alive; just return ctx errors.
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Scheduler) dispatchDue(ctx context.Context) error {
	now := time.Now().UTC()

	var due []*ScheduledJob

	s.mu.Lock()
	for _, j := range s.jobs {
		if !j.Active {
			continue
		}
		if j.NextRun.IsZero() {
			j.NextRun = j.Schedule.Next(now)
		}
		if !j.NextRun.IsZero() && !j.NextRun.After(now) && !j.Running {
			j.Running = true
			due = append(due, j)
		}
	}
	s.mu.Unlock()

	for _, job := range due {
		go s.runJob(ctx, job)
	}

	return nil
}

func (s *Scheduler) runJob(ctx context.Context, job *ScheduledJob) {
	defer func() {
		s.mu.Lock()
		job.Running = false
		job.NextRun = job.Schedule.Next(time.Now().UTC())
		if job.NextRun.IsZero() {
			job.Active = false
		}
		s.mu.Unlock()
	}()

	if s.runner == nil {
		return
	}

	_, _ = s.runner.RunOnce(ctx, job.StackName, job.Job, job.ConfigHash)
}

func (s *Scheduler) getInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.interval <= 0 {
		return 5 * time.Second
	}
	return s.interval
}

func jobKey(stackName, jobName string) string {
	return stackName + ":" + jobName
}

func (s *Scheduler) List() []ScheduledJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]ScheduledJob, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, *j)
	}
	return out
}