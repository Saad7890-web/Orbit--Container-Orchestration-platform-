package controller

import (
	"context"
	"time"

	"github.com/Saad7890-web/orbit/internal/docker"
	"github.com/Saad7890-web/orbit/internal/health"
	"github.com/Saad7890-web/orbit/internal/models"
)

type ServiceLifecycleManager struct {
	monitor *health.Monitor
}

func NewServiceLifecycleManager(runtime docker.Runtime, repo health.ServiceRepository, interval time.Duration) *ServiceLifecycleManager {
	return &ServiceLifecycleManager{
		monitor: health.NewMonitor(runtime, repo, interval),
	}
}

func (m *ServiceLifecycleManager) Watch(ctx context.Context, stackName string, svc models.Service, configHash string) error {
	return m.monitor.Run(ctx, health.ServiceInstance{
		StackName:     stackName,
		ContainerName: docker.BuildContainerName(stackName, svc.Name),
		Service:       svc,
		ConfigHash:    configHash,
	})
}

func (m *ServiceLifecycleManager) ObserveOnce(ctx context.Context, stackName string, svc models.Service, configHash string) error {
	return m.monitor.ObserveOnce(ctx, health.ServiceInstance{
		StackName:     stackName,
		ContainerName: docker.BuildContainerName(stackName, svc.Name),
		Service:       svc,
		ConfigHash:    configHash,
	})
}