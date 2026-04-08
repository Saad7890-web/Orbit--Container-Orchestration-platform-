package models

import (
	"fmt"
	"strings"
)

type WorkloadKind string

const (
	WorkloadKindService  WorkloadKind = "service"
	WorkloadKindJob      WorkloadKind = "job"
	WorkloadKindTrigger  WorkloadKind = "trigger"
	WorkloadKindStack    WorkloadKind = "stack"
)

func (k WorkloadKind) Valid() bool {
	switch k {
	case WorkloadKindService, WorkloadKindJob, WorkloadKindTrigger, WorkloadKindStack:
		return true
	default:
		return false
	}
}

type HealthStatus string

const (
	HealthUnknown  HealthStatus = "unknown"
	HealthHealthy  HealthStatus = "healthy"
	HealthDegraded HealthStatus = "degraded"
	HealthUnhealthy HealthStatus = "unhealthy"
)

func (s HealthStatus) Valid() bool {
	switch s {
	case HealthUnknown, HealthHealthy, HealthDegraded, HealthUnhealthy:
		return true
	default:
		return false
	}
}

type ExecutionStatus string

const (
	ExecutionPending  ExecutionStatus = "pending"
	ExecutionRunning  ExecutionStatus = "running"
	ExecutionSucceeded ExecutionStatus = "succeeded"
	ExecutionFailed   ExecutionStatus = "failed"
	ExecutionSkipped  ExecutionStatus = "skipped"
	ExecutionCanceled ExecutionStatus = "canceled"
)

func (s ExecutionStatus) Valid() bool {
	switch s {
	case ExecutionPending, ExecutionRunning, ExecutionSucceeded, ExecutionFailed, ExecutionSkipped, ExecutionCanceled:
		return true
	default:
		return false
	}
}

type LifecycleStatus string

const (
	LifecycleUnknown  LifecycleStatus = "unknown"
	LifecycleStarting  LifecycleStatus = "starting"
	LifecycleRunning   LifecycleStatus = "running"
	LifecycleStopping  LifecycleStatus = "stopping"
	LifecycleStopped   LifecycleStatus = "stopped"
	LifecycleRestarting LifecycleStatus = "restarting"
	LifecycleFailed    LifecycleStatus = "failed"
)

func (s LifecycleStatus) Valid() bool {
	switch s {
	case LifecycleUnknown, LifecycleStarting, LifecycleRunning, LifecycleStopping, LifecycleStopped, LifecycleRestarting, LifecycleFailed:
		return true
	default:
		return false
	}
}

type RestartPolicy string

const (
	RestartPolicyNo           RestartPolicy = "no"
	RestartPolicyAlways       RestartPolicy = "always"
	RestartPolicyOnFailure    RestartPolicy = "on-failure"
	RestartPolicyUnlessStopped RestartPolicy = "unless-stopped"
)

func (p RestartPolicy) Valid() bool {
	switch p {
	case RestartPolicyNo, RestartPolicyAlways, RestartPolicyOnFailure, RestartPolicyUnlessStopped:
		return true
	default:
		return false
	}
}

func NormalizeName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

func ErrInvalidEnum(field, value string) error {
	return fmt.Errorf("invalid %s: %q", field, value)
}