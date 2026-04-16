package docker

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/Saad7890-web/orbit/internal/models"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

func BuildCreateConfig(spec ContainerSpec, configHash string) (container.Config, container.HostConfig, network.NetworkingConfig, error) {
	if strings.TrimSpace(spec.Image) == "" {
		return container.Config{}, container.HostConfig{}, network.NetworkingConfig{}, fmt.Errorf("image is required")
	}
	if strings.TrimSpace(spec.Name) == "" {
		return container.Config{}, container.HostConfig{}, network.NetworkingConfig{}, fmt.Errorf("container name is required")
	}

	cmd := append([]string{}, spec.Command...)
	if len(spec.WorkingDir) > 0 {
		_ = spec.WorkingDir
	}

	cfg := container.Config{
		Image:      spec.Image,
		Cmd:        cmd,
		WorkingDir: spec.WorkingDir,
		Env:        SortEnv(spec.Env),
		Labels:     WorkloadLabels(spec.StackName, spec.Kind, spec.Workload, configHash, spec.Labels),
		User:       spec.User,
	}

	if len(spec.Command) == 0 {
		cfg.Cmd = nil
	}

	hostCfg := container.HostConfig{
		NetworkMode:  container.NetworkMode(spec.NetworkMode),
		RestartPolicy: restartPolicy(spec.Restart),
	}

	if len(spec.Volumes) > 0 {
		mounts := make([]mountSpec, 0, len(spec.Volumes))
		for _, v := range spec.Volumes {
			mounts = append(mounts, mountSpec{HostPath: v.HostPath, ContainerPath: v.ContainerPath, ReadOnly: v.ReadOnly})
		}
		var err error
		hostCfg.Mounts, err = buildMounts(mounts)
		if err != nil {
			return container.Config{}, container.HostConfig{}, network.NetworkingConfig{}, err
		}
	}

	exposed, bindings, err := buildPortBindings(spec.Ports)
	if err != nil {
		return container.Config{}, container.HostConfig{}, network.NetworkingConfig{}, err
	}
	cfg.ExposedPorts = exposed
	hostCfg.PortBindings = bindings

	if spec.HealthCheck != nil {
		hc, err := buildHealthCheck(spec.HealthCheck)
		if err != nil {
			return container.Config{}, container.HostConfig{}, network.NetworkingConfig{}, err
		}
		cfg.Healthcheck = hc
	}

	return cfg, hostCfg, network.NetworkingConfig{}, nil
}

type mountSpec struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

func buildMounts(items []mountSpec) ([]mount.Mount, error) {
	out := make([]mount.Mount, 0, len(items))

	for _, m := range items {
		if strings.TrimSpace(m.HostPath) == "" {
			return nil, fmt.Errorf("volume hostPath is required")
		}
		if strings.TrimSpace(m.ContainerPath) == "" {
			return nil, fmt.Errorf("volume containerPath is required")
		}
		if !strings.HasPrefix(m.ContainerPath, "/") {
			return nil, fmt.Errorf("containerPath must be absolute: %q", m.ContainerPath)
		}

		out = append(out, mount.Mount{
			Type:     mount.TypeBind,
			Source:   m.HostPath,
			Target:   m.ContainerPath,
			ReadOnly: m.ReadOnly,
		})
	}

	return out, nil
}

func buildPortBindings(ports []models.PortMapping) (nat.PortSet, nat.PortMap, error) {
	exposed := nat.PortSet{}
	bindings := nat.PortMap{}

	for _, p := range ports {
		if p.HostPort < 1 || p.HostPort > 65535 {
			return nil, nil, fmt.Errorf("invalid hostPort %d", p.HostPort)
		}
		if p.ContainerPort < 1 || p.ContainerPort > 65535 {
			return nil, nil, fmt.Errorf("invalid containerPort %d", p.ContainerPort)
		}
		if p.HostIP != "" && net.ParseIP(p.HostIP) == nil {
			return nil, nil, fmt.Errorf("invalid hostIP %q", p.HostIP)
		}
		proto := strings.ToLower(p.Protocol)
		if proto == "" {
			proto = "tcp"
		}
		if proto != "tcp" && proto != "udp" {
			return nil, nil, fmt.Errorf("invalid protocol %q", p.Protocol)
		}

		port := nat.Port(fmt.Sprintf("%d/%s", p.ContainerPort, proto))
		exposed[port] = struct{}{}
		bindings[port] = append(bindings[port], nat.PortBinding{
			HostIP:   p.HostIP,
			HostPort: strconv.Itoa(p.HostPort),
		})
	}

	return exposed, bindings, nil
}

func buildHealthCheck(h *models.HealthCheck) (*container.HealthConfig, error) {
	switch h.Kind {
	case "http":
		if h.Port < 1 || h.Port > 65535 {
			return nil, fmt.Errorf("invalid healthcheck port %d", h.Port)
		}
		if strings.TrimSpace(h.Path) == "" {
			return nil, fmt.Errorf("healthcheck path is required for http")
		}
		test := []string{
			"CMD-SHELL",
			fmt.Sprintf(`wget -qO- http://127.0.0.1:%d%s >/dev/null 2>&1 || exit 1`, h.Port, h.Path),
		}
		return &container.HealthConfig{
			Test:          test,
			Interval:      durationOrDefault(h.IntervalSeconds, 30*time.Second),
			Timeout:       durationOrDefault(h.TimeoutSeconds, 5*time.Second),
			Retries:       h.Retries,
			StartPeriod:   time.Duration(h.StartPeriodSeconds) * time.Second,
			StartInterval: 0,
		}, nil

	case "tcp":
		if h.Port < 1 || h.Port > 65535 {
			return nil, fmt.Errorf("invalid healthcheck port %d", h.Port)
		}
		test := []string{
			"CMD-SHELL",
			fmt.Sprintf(`nc -z 127.0.0.1 %d || exit 1`, h.Port),
		}
		return &container.HealthConfig{
			Test:          test,
			Interval:      durationOrDefault(h.IntervalSeconds, 30*time.Second),
			Timeout:       durationOrDefault(h.TimeoutSeconds, 5*time.Second),
			Retries:       h.Retries,
			StartPeriod:   time.Duration(h.StartPeriodSeconds) * time.Second,
			StartInterval: 0,
		}, nil

	case "cmd":
		if strings.TrimSpace(h.Command) == "" {
			return nil, fmt.Errorf("healthcheck command is required for cmd")
		}
		test := []string{"CMD-SHELL", h.Command}
		return &container.HealthConfig{
			Test:          test,
			Interval:      durationOrDefault(h.IntervalSeconds, 30*time.Second),
			Timeout:       durationOrDefault(h.TimeoutSeconds, 5*time.Second),
			Retries:       h.Retries,
			StartPeriod:   time.Duration(h.StartPeriodSeconds) * time.Second,
			StartInterval: 0,
		}, nil

	default:
		return nil, fmt.Errorf("invalid healthcheck kind %q", h.Kind)
	}
}

func durationOrDefault(seconds int, fallback time.Duration) time.Duration {
	if seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func restartPolicy(p models.RestartPolicy) container.RestartPolicy {
	switch p {
	case models.RestartPolicyAlways:
		return container.RestartPolicy{
			Name: container.RestartPolicyMode(p),
		}

	case models.RestartPolicyOnFailure:
		return container.RestartPolicy{
			Name:              container.RestartPolicyMode(p),
			MaximumRetryCount: 3,
		}

	case models.RestartPolicyUnlessStopped:
		return container.RestartPolicy{
			Name: container.RestartPolicyMode(p),
		}

	default:
		return container.RestartPolicy{
			Name: container.RestartPolicyMode(models.RestartPolicyNo),
		}
	}
}