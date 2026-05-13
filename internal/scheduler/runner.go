package scheduler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Saad7890-web/orbit/internal/docker"
	"github.com/Saad7890-web/orbit/internal/models"
)

type JobRepository interface {
	UpsertJob(ctx context.Context, stackName string, job models.Job, configHash string, lastStatus models.ExecutionStatus, lastError string, lastRunAt *time.Time) error
	SaveExecution(ctx context.Context, e models.Execution) error
}

type JobRunner struct {
	runtime docker.Runtime
	repo    JobRepository
}

func NewJobRunner(runtime docker.Runtime, repo JobRepository) *JobRunner {
	return &JobRunner{
		runtime: runtime,
		repo:    repo,
	}
}

func (r *JobRunner) RunOnce(ctx context.Context, stackName string, job models.Job, configHash string) (*models.Execution, error) {
	if r == nil {
		return nil, fmt.Errorf("job runner is nil")
	}
	if r.runtime == nil {
		return nil, fmt.Errorf("docker runtime is nil")
	}
	if r.repo == nil {
		return nil, fmt.Errorf("repository is nil")
	}
	if err := job.Validate(); err != nil {
		return nil, err
	}

	execID, err := newExecutionID()
	if err != nil {
		return nil, err
	}

	execution := &models.Execution{
		ID:          execID,
		Kind:        models.WorkloadKindJob,
		Workload:    job.Name,
		Status:      models.ExecutionRunning,
		StartedAt:   time.Now().UTC(),
		TriggerName: "",
		LogsRef:     "",
	}

	containerName := docker.BuildContainerName(stackName, job.Name) + "-" + shortID(execID)

	spec := docker.ContainerSpec{
		Name:       containerName,
		StackName:  stackName,
		Kind:       models.WorkloadKindJob,
		Workload:   job.Name,
		Image:      job.Image,
		Command:    job.Command,
		WorkingDir: job.WorkingDir,
		Env:        job.Env,
		Labels:     job.Labels,
		Volumes:    job.Volumes,
	}

	spec.Labels = docker.WorkloadLabels(stackName, models.WorkloadKindJob, job.Name, configHash, job.Labels)

	if job.Timeout != "" {
		timeout, err := time.ParseDuration(job.Timeout)
		if err != nil {
			return nil, fmt.Errorf("parse job timeout: %w", err)
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	containerID, err := r.runtime.CreateContainer(ctx, spec)
	if err != nil {
		execution.Status = models.ExecutionFailed
		execution.Error = err.Error()
		_ = r.repo.SaveExecution(ctx, *execution)
		_ = r.repo.UpsertJob(ctx, stackName, job, configHash, models.ExecutionFailed, err.Error(), &execution.StartedAt)
		return execution, err
	}
	execution.LogsRef = containerID

	defer func() {
		_ = r.runtime.RemoveContainer(context.Background(), containerID, true, true)
	}()

	if err := r.runtime.StartContainer(ctx, containerID); err != nil {
		execution.Status = models.ExecutionFailed
		execution.Error = err.Error()
		ended := time.Now().UTC()
		execution.EndedAt = &ended
		_ = r.repo.SaveExecution(ctx, *execution)
		_ = r.repo.UpsertJob(ctx, stackName, job, configHash, models.ExecutionFailed, err.Error(), &execution.StartedAt)
		return execution, err
	}

	exitCode, waitErr := r.runtime.WaitContainer(ctx, containerID)
	ended := time.Now().UTC()
	execution.EndedAt = &ended

	if waitErr != nil {
		execution.Status = models.ExecutionFailed
		execution.Error = waitErr.Error()
		_ = r.repo.SaveExecution(ctx, *execution)
		_ = r.repo.UpsertJob(ctx, stackName, job, configHash, models.ExecutionFailed, waitErr.Error(), &ended)
		return execution, waitErr
	}

	code := int(exitCode)
	execution.ExitCode = &code
	if exitCode == 0 {
		execution.Status = models.ExecutionSucceeded
		execution.Error = ""
		_ = r.repo.SaveExecution(ctx, *execution)
		_ = r.repo.UpsertJob(ctx, stackName, job, configHash, models.ExecutionSucceeded, "", &ended)
		return execution, nil
	}

	execution.Status = models.ExecutionFailed
	execution.Error = fmt.Sprintf("job exited with code %d", exitCode)
	_ = r.repo.SaveExecution(ctx, *execution)
	_ = r.repo.UpsertJob(ctx, stackName, job, configHash, models.ExecutionFailed, execution.Error, &ended)

	return execution, fmt.Errorf(execution.Error)
}

func newExecutionID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}