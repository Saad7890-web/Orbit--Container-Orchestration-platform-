package models

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
)

type PortMapping struct {
	HostIP        string `json:"hostIP,omitempty" yaml:"hostIP,omitempty"`
	HostPort      int    `json:"hostPort" yaml:"hostPort"`
	ContainerPort int    `json:"containerPort" yaml:"containerPort"`
	Protocol      string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

func (p PortMapping) Validate() error {
	if p.HostPort < 1 || p.HostPort > 65535 {
		return fmt.Errorf("invalid hostPort %d", p.HostPort)
	}
	if p.ContainerPort < 1 || p.ContainerPort > 65535 {
		return fmt.Errorf("invalid containerPort %d", p.ContainerPort)
	}
	if p.HostIP != "" && net.ParseIP(p.HostIP) == nil {
		return fmt.Errorf("invalid hostIP %q", p.HostIP)
	}
	switch p.Protocol {
	case "", "tcp", "udp":
		return nil
	default:
		return fmt.Errorf("invalid protocol %q", p.Protocol)
	}
}

type VolumeMount struct {
	HostPath      string `json:"hostPath" yaml:"hostPath"`
	ContainerPath  string `json:"containerPath" yaml:"containerPath"`
	ReadOnly      bool   `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
}

func (v VolumeMount) Validate() error {
	if strings.TrimSpace(v.HostPath) == "" {
		return errors.New("volume hostPath is required")
	}
	if strings.TrimSpace(v.ContainerPath) == "" {
		return errors.New("volume containerPath is required")
	}
	if !strings.HasPrefix(v.ContainerPath, "/") {
		return fmt.Errorf("containerPath must be absolute: %q", v.ContainerPath)
	}
	return nil
}


type EnvVar struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

type HealthCheck struct {
	Kind                string `json:"kind" yaml:"kind"` // http, tcp, cmd
	Path                string `json:"path,omitempty" yaml:"path,omitempty"`
	Port                int    `json:"port,omitempty" yaml:"port,omitempty"`
	Command             string `json:"command,omitempty" yaml:"command,omitempty"`
	IntervalSeconds     int    `json:"intervalSeconds,omitempty" yaml:"intervalSeconds,omitempty"`
	TimeoutSeconds      int    `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`
	Retries             int    `json:"retries,omitempty" yaml:"retries,omitempty"`
	StartPeriodSeconds  int    `json:"startPeriodSeconds,omitempty" yaml:"startPeriodSeconds,omitempty"`
}

func (h HealthCheck) Validate() error {
	switch h.Kind {
	case "http":
		if strings.TrimSpace(h.Path) == "" {
			return errors.New("healthcheck path is required for http")
		}
		if h.Port < 1 || h.Port > 65535 {
			return fmt.Errorf("invalid healthcheck port %d", h.Port)
		}
	case "tcp":
		if h.Port < 1 || h.Port > 65535 {
			return fmt.Errorf("invalid healthcheck port %d", h.Port)
		}
	case "cmd":
		if strings.TrimSpace(h.Command) == "" {
			return errors.New("healthcheck command is required for cmd")
		}
	default:
		return fmt.Errorf("invalid healthcheck kind %q", h.Kind)
	}

	if h.IntervalSeconds < 0 || h.TimeoutSeconds < 0 || h.Retries < 0 || h.StartPeriodSeconds < 0 {
		return errors.New("healthcheck timing fields must be non-negative")
	}
	return nil
}


type Service struct {
	Name         string            `json:"name" yaml:"name"`
	Image        string            `json:"image" yaml:"image"`
	Command      []string          `json:"command,omitempty" yaml:"command,omitempty"`
	Args         []string          `json:"args,omitempty" yaml:"args,omitempty"`
	WorkingDir   string            `json:"workingDir,omitempty" yaml:"workingDir,omitempty"`
	Env          map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Ports        []PortMapping     `json:"ports,omitempty" yaml:"ports,omitempty"`
	Volumes      []VolumeMount     `json:"volumes,omitempty" yaml:"volumes,omitempty"`
	Labels       map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	DependsOn    []string          `json:"dependsOn,omitempty" yaml:"dependsOn,omitempty"`
	Restart      RestartPolicy     `json:"restart,omitempty" yaml:"restart,omitempty"`
	HealthCheck  *HealthCheck      `json:"healthCheck,omitempty" yaml:"healthCheck,omitempty"`
	Enabled      bool              `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

var serviceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

func (s Service) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("service name is required")
	}
	if !serviceNamePattern.MatchString(s.Name) {
		return fmt.Errorf("invalid service name %q", s.Name)
	}
	if strings.TrimSpace(s.Image) == "" {
		return errors.New("service image is required")
	}
	if s.Restart != "" && !s.Restart.Valid() {
		return ErrInvalidEnum("restart policy", string(s.Restart))
	}
	for i := range s.Ports {
		if err := s.Ports[i].Validate(); err != nil {
			return fmt.Errorf("ports[%d]: %w", i, err)
		}
	}
	for i := range s.Volumes {
		if err := s.Volumes[i].Validate(); err != nil {
			return fmt.Errorf("volumes[%d]: %w", i, err)
		}
	}
	if s.HealthCheck != nil {
		if err := s.HealthCheck.Validate(); err != nil {
			return fmt.Errorf("healthCheck: %w", err)
		}
	}
	for _, dep := range s.DependsOn {
		if strings.TrimSpace(dep) == "" {
			return errors.New("dependsOn contains an empty service name")
		}
	}
	return nil
}
