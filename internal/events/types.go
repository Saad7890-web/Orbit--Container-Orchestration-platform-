package events

import (
	"time"

	"github.com/Saad7890-web/orbit/internal/models"
)

type Type string

const (
	TypeConfigApplied   Type = "config.applied"
	TypeStackReconciled Type = "stack.reconciled"

	TypeServiceStarted   Type = "service.started"
	TypeServiceStopped   Type = "service.stopped"
	TypeServiceRestarted Type = "service.restarted"
	TypeServiceFailed    Type = "service.failed"
	TypeServiceHealthy   Type = "service.healthy"
	TypeServiceUnhealthy Type = "service.unhealthy"

	TypeJobScheduled  Type = "job.scheduled"
	TypeJobStarted    Type = "job.started"
	TypeJobCompleted  Type = "job.completed"
	TypeJobFailed     Type = "job.failed"

	TypeTriggerFired  Type = "trigger.fired"
	TypeTriggerMatched Type = "trigger.matched"

	TypeExecutionCreated Type = "execution.created"
	TypeExecutionEnded   Type = "execution.ended"

	TypeHealthChanged Type = "health.changed"
	TypeLogLine       Type = "log.line"
	TypeError         Type = "error"
)

func (t Type) String() string { return string(t) }

type Event struct {
	ID        string            `json:"id"`
	Type      Type              `json:"type"`
	Source    string            `json:"source"`
	StackName string            `json:"stackName,omitempty"`
	Workload  string            `json:"workload,omitempty"`
	Kind      models.WorkloadKind `json:"kind,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Data      map[string]any    `json:"data,omitempty"`
}

type HandlerFunc func(Event) error

type Subscription struct {
	ID   int
	Events <-chan Event
}