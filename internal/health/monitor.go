package health

import (
	"context"
	"fmt"
	"time"

	"github.com/Saad7890-web/orbit/internal/docker"
	"github.com/Saad7890-web/orbit/internal/models"
)

type ServiceRepository interface {
	UpsertService(ctx context.Context, stackName string, svc models.Service, configHash string, status models.LifecycleStatus, health models.HealthStatus, lastError string) error
}

type ServiceRuntime interface {
	InspectContainer(ctx context.Context, id string) (*docker.ContainerInfo, error)
	StartContainer(ctx context.Context, id string) error
	RestartContainer(ctx context.Context, id string, timeout time.Duration) error
	StopContainer(ctx context.Context, id string, timeout time.Duration) error
}

type ServiceInstance struct {
	StackName    string
	ContainerName string
	Service      models.Service
	ConfigHash   string
}

type Monitor struct {
	runtime        ServiceRuntime
	repo           ServiceRepository
	interval       time.Duration
	restartTimeout time.Duration
	baseBackoff    time.Duration
	maxBackoff     time.Duration
}

func NewMonitor(runtime ServiceRuntime, repo ServiceRepository, interval time.Duration) *Monitor {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &Monitor{
		runtime:        runtime,
		repo:           repo,
		interval:       interval,
		restartTimeout: 10 * time.Second,
		baseBackoff:    5 * time.Second,
		maxBackoff:     60 * time.Second,
	}
}

func (m *Monitor) Run(ctx context.Context, instance ServiceInstance) error {
	if m == nil {
		return fmt.Errorf("monitor is nil")
	}
	if m.runtime == nil {
		return fmt.Errorf("runtime is nil")
	}
	if m.repo == nil {
		return fmt.Errorf("repository is nil")
	}
	if instance.StackName == "" || instance.ContainerName == "" {
		return fmt.Errorf("service instance is incomplete")
	}

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	attempts := 0

	// Run one immediate check on startup.
	_ = m.observe(ctx, instance, &attempts)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			_ = m.observe(ctx, instance, &attempts)
		}
	}
}

func (m *Monitor) ObserveOnce(ctx context.Context, instance ServiceInstance) error {
	if m == nil {
		return fmt.Errorf("monitor is nil")
	}
	attempts := 0
	return m.observe(ctx, instance, &attempts)
}

func (m *Monitor) observe(ctx context.Context, instance ServiceInstance, attempts *int) error {
	info, err := m.runtime.InspectContainer(ctx, instance.ContainerName)
	if err != nil {
		_ = m.repo.UpsertService(
			ctx,
			instance.StackName,
			instance.Service,
			instance.ConfigHash,
			models.LifecycleFailed,
			models.HealthUnknown,
			err.Error(),
		)
		if attempts != nil {
			*attempts++
		}
		return err
	}

	lifecycle := LifecycleStatusFromInfo(info)
	health := HealthStatusFromInfo(info)

	lastErr := ""
	if lifecycle == models.LifecycleFailed && info.ExitCode != nil && *info.ExitCode != 0 {
		lastErr = fmt.Sprintf("container exited with code %d", *info.ExitCode)
	}
	if health == models.HealthUnhealthy {
		lastErr = "container healthcheck reported unhealthy"
	}

	if err := m.repo.UpsertService(
		ctx,
		instance.StackName,
		instance.Service,
		instance.ConfigHash,
		lifecycle,
		health,
		lastErr,
	); err != nil {
		return err
	}

	if !ShouldRestart(instance.Service.Restart, info) {
		if attempts != nil && lifecycle == models.LifecycleRunning && health != models.HealthUnhealthy {
			*attempts = 0
		}
		return nil
	}

	delay := BackoffDelay(*attempts, m.baseBackoff, m.maxBackoff)
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	restartErr := m.restart(ctx, instance, info)
	if restartErr != nil {
		if attempts != nil {
			*attempts++
		}
		_ = m.repo.UpsertService(
			ctx,
			instance.StackName,
			instance.Service,
			instance.ConfigHash,
			models.LifecycleFailed,
			models.HealthUnknown,
			restartErr.Error(),
		)
		return restartErr
	}

	if attempts != nil {
		*attempts = 0
	}

	// Capture the post-restart state.
	postInfo, postErr := m.runtime.InspectContainer(ctx, instance.ContainerName)
	if postErr != nil {
		_ = m.repo.UpsertService(
			ctx,
			instance.StackName,
			instance.Service,
			instance.ConfigHash,
			models.LifecycleFailed,
			models.HealthUnknown,
			postErr.Error(),
		)
		return postErr
	}

	return m.repo.UpsertService(
		ctx,
		instance.StackName,
		instance.Service,
		instance.ConfigHash,
		LifecycleStatusFromInfo(postInfo),
		HealthStatusFromInfo(postInfo),
		"",
	)
}

func (m *Monitor) restart(ctx context.Context, instance ServiceInstance, info *docker.ContainerInfo) error {
	if info != nil && info.State == "running" {
		return m.runtime.RestartContainer(ctx, instance.ContainerName, m.restartTimeout)
	}
	return m.runtime.StartContainer(ctx, instance.ContainerName)
}