package controller

import (
	"context"
	"fmt"

	"github.com/Saad7890-web/orbit/internal/docker"
	"github.com/Saad7890-web/orbit/internal/models"
)

type Repository interface {
	SaveStack(ctx context.Context, stack models.Stack, configHash string) error
	UpsertService(ctx context.Context, stackName string, svc models.Service, configHash string, status models.LifecycleStatus, health models.HealthStatus, lastError string) error
	UpsertJob(ctx context.Context, stackName string, job models.Job, configHash string, lastStatus models.ExecutionStatus, lastError string, lastRunAt *funcTime) error
	UpsertTrigger(ctx context.Context, stackName string, trigger models.Trigger, configHash string, lastStatus models.ExecutionStatus, lastError string, lastFiredAt *funcTime) error
	SetMetadata(ctx context.Context, key, value string) error
}

type funcTime = interface {
	UTC() string
}

type Controller struct {
	repo   Repository
	runtime docker.Runtime
}

type Options struct {
	PruneRemovedServices bool
}

func New(repo Repository, runtime docker.Runtime) *Controller {
	return &Controller{
		repo:   repo,
		runtime: runtime,
	}
}

func (c *Controller) Apply(ctx context.Context, stack models.Stack, configHash string) (*Report, error) {
	if c == nil {
		return nil, fmt.Errorf("controller is nil")
	}
	if c.repo == nil {
		return nil, fmt.Errorf("repository is nil")
	}
	if c.runtime == nil {
		return nil, fmt.Errorf("docker runtime is nil")
	}
	if err := stack.Validate(); err != nil {
		return nil, err
	}

	desired := NewDesiredState(stack, configHash)
	reconciler := NewReconciler(c.repo, c.runtime)

	return reconciler.Reconcile(ctx, desired)
}