package models

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type Execution struct {
	ID          string          `json:"id" yaml:"id"`
	Kind        WorkloadKind    `json:"kind" yaml:"kind"`
	Workload    string          `json:"workload" yaml:"workload"`
	TriggerName string          `json:"triggerName,omitempty" yaml:"triggerName,omitempty"`
	Status      ExecutionStatus `json:"status" yaml:"status"`
	StartedAt   time.Time       `json:"startedAt" yaml:"startedAt"`
	EndedAt     *time.Time      `json:"endedAt,omitempty" yaml:"endedAt,omitempty"`
	ExitCode    *int            `json:"exitCode,omitempty" yaml:"exitCode,omitempty"`
	Error       string          `json:"error,omitempty" yaml:"error,omitempty"`
	LogsRef     string          `json:"logsRef,omitempty" yaml:"logsRef,omitempty"`
}

func (e Execution) Validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return errors.New("execution id is required")
	}
	if !e.Kind.Valid() {
		return ErrInvalidEnum("workload kind", string(e.Kind))
	}
	if strings.TrimSpace(e.Workload) == "" {
		return errors.New("execution workload is required")
	}
	if !e.Status.Valid() {
		return ErrInvalidEnum("execution status", string(e.Status))
	}
	if !e.EndedAt.IsZero() && e.EndedAt.Before(e.StartedAt) {
		return errors.New("endedAt cannot be before startedAt")
	}
	if e.ExitCode != nil && (*e.ExitCode < 0 || *e.ExitCode > 255) {
		return fmt.Errorf("invalid exit code %d", *e.ExitCode)
	}
	return nil
}

type ExecutionSummary struct {
	WorkloadKind WorkloadKind    `json:"workloadKind" yaml:"workloadKind"`
	WorkloadName string          `json:"workloadName" yaml:"workloadName"`
	LastStatus   ExecutionStatus `json:"lastStatus" yaml:"lastStatus"`
	LastRunAt    *time.Time      `json:"lastRunAt,omitempty" yaml:"lastRunAt,omitempty"`
	LastError    string          `json:"lastError,omitempty" yaml:"lastError,omitempty"`
}