package docker

import (
	"context"
	"io"
	"time"

	"github.com/Saad7890-web/orbit/internal/models"
)

const (
	LabelManaged  = "orbit.managed"
	LabelStack    = "orbit.stack"
	LabelKind     = "orbit.kind"
	LabelWorkload = "orbit.workload"
	LabelHash     = "orbit.configHash"
)

type Runtime interface {
	Ping(ctx context.Context) error
	Close() error

	EnsureImage(ctx context.Context, image string) error

	CreateContainer(ctx context.Context, spec ContainerSpec) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string, timeout time.Duration) error
	RestartContainer(ctx context.Context, id string, timeout time.Duration) error
	RemoveContainer(ctx context.Context, id string, force bool, removeVolumes bool) error

	InspectContainer(ctx context.Context, id string) (*ContainerInfo, error)
	ListContainers(ctx context.Context, filter LabelFilter) ([]ContainerSummary, error)
	ContainerLogs(ctx context.Context, id string, opts LogOptions) (io.ReadCloser, error)
	WaitContainer(ctx context.Context, id string) (int64, error)
}

type Options struct {
	Host        string
	UseEnv      bool
	APIVersion  bool
	PingTimeout time.Duration
}

type LabelFilter map[string]string

type ContainerSpec struct {
	Name        string
	StackName   string
	Kind        models.WorkloadKind
	Workload    string
	Image       string
	Command     []string
	WorkingDir  string
	Env         map[string]string
	Labels      map[string]string
	Ports       []models.PortMapping
	Volumes     []models.VolumeMount
	Restart     models.RestartPolicy
	HealthCheck *models.HealthCheck
	User        string
	NetworkMode string
}

type LogOptions struct {
	Stdout     bool
	Stderr     bool
	Follow     bool
	Timestamps bool
	Tail       string
	Since      time.Time
}

type ContainerSummary struct {
	ID        string
	Name      string
	Image     string
	State     string
	Status    string
	Labels    map[string]string
	CreatedAt time.Time
}

type ContainerInfo struct {
	ID        string
	Name      string
	Image     string
	State     string
	Status    string
	ExitCode  *int
	StartedAt *time.Time
	FinishedAt *time.Time
	Health    string
	Labels    map[string]string
}