package health

import (
	"fmt"
	"time"

	"github.com/Saad7890-web/orbit/internal/docker"
	"github.com/Saad7890-web/orbit/internal/models"
)

func LifecycleStatusFromInfo(info *docker.ContainerInfo) models.LifecycleStatus {
	if info == nil {
		return models.LifecycleUnknown
	}

	switch info.State {
	case "running":
		return models.LifecycleRunning
	case "restarting":
		return models.LifecycleRestarting
	case "created":
		return models.LifecycleStarting
	case "paused":
		return models.LifecycleStopped
	case "exited":
		return models.LifecycleStopped
	case "dead":
		return models.LifecycleFailed
	default:
		return models.LifecycleUnknown
	}
}

func HealthStatusFromInfo(info *docker.ContainerInfo) models.HealthStatus {
	if info == nil {
		return models.HealthUnknown
	}

	switch info.Health {
	case "healthy":
		return models.HealthHealthy
	case "starting":
		return models.HealthDegraded
	case "unhealthy":
		return models.HealthUnhealthy
	case "":
		if info.State == "running" {
			return models.HealthUnknown
		}
		return models.HealthUnknown
	default:
		return models.HealthUnknown
	}
}

func ShouldRestart(policy models.RestartPolicy, info *docker.ContainerInfo) bool {
	if info == nil {
		return false
	}

	lifecycle := LifecycleStatusFromInfo(info)
	health := HealthStatusFromInfo(info)

	switch policy {
	case models.RestartPolicyNo:
		return false

	case models.RestartPolicyAlways:
		return lifecycle != models.LifecycleRunning || health == models.HealthUnhealthy

	case models.RestartPolicyOnFailure:
		if lifecycle == models.LifecycleFailed || lifecycle == models.LifecycleStopped {
			return info.ExitCode != nil && *info.ExitCode != 0
		}
		return health == models.HealthUnhealthy

	case models.RestartPolicyUnlessStopped:
		if lifecycle == models.LifecycleStopped || lifecycle == models.LifecycleFailed {
			return true
		}
		return health == models.HealthUnhealthy

	default:
		return false
	}
}

func BackoffDelay(attempt int, base, max time.Duration) time.Duration {
	if attempt <= 0 {
		if base > 0 {
			return base
		}
		return 5 * time.Second
	}
	if base <= 0 {
		base = 5 * time.Second
	}
	if max <= 0 {
		max = 60 * time.Second
	}

	delay := base
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= max {
			return max
		}
	}
	if delay > max {
		return max
	}
	return delay
}

func SnapshotError(prefix string, err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%s: %v", prefix, err)
}