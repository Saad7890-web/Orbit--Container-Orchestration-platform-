package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

type Client struct {
	cli *client.Client
}

func New(ctx context.Context, opts Options) (*Client, error) {
	var (
		cli *client.Client
		err error
	)

	switch {
	case opts.UseEnv || strings.TrimSpace(opts.Host) == "":
		cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	default:
		cli, err = client.NewClientWithOpts(
			client.WithHost(opts.Host),
			client.WithAPIVersionNegotiation(),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	c := &Client{cli: cli}
	pingCtx := ctx
	cancel := func() {}
	if opts.PingTimeout > 0 {
		pingCtx, cancel = context.WithTimeout(ctx, opts.PingTimeout)
	}
	defer cancel()

	if err := c.Ping(pingCtx); err != nil {
		_ = c.Close()
		return nil, err
	}

	return c, nil
}

func (c *Client) Close() error {
	if c == nil || c.cli == nil {
		return nil
	}
	return c.cli.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.cli.Ping(ctx)
	if err != nil {
		return fmt.Errorf("docker ping failed: %w", err)
	}
	return nil
}

func (c *Client) EnsureImage(ctx context.Context, imageName string) error {
	_, _, err := c.cli.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		return nil
	}

	rc, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %q: %w", imageName, err)
	}
	defer rc.Close()

	_, _ = io.Copy(io.Discard, rc)
	return nil
}

func (c *Client) CreateContainer(ctx context.Context, spec ContainerSpec) (string, error) {
	if err := c.EnsureImage(ctx, spec.Image); err != nil {
		return "", err
	}

	configHash := spec.Labels[LabelHash]
	createCfg, hostCfg, netCfg, err := BuildCreateConfig(spec, configHash)
	if err != nil {
		return "", err
	}

	resp, err := c.cli.ContainerCreate(ctx, &createCfg, &hostCfg, &netCfg, nil, spec.Name)
	if err != nil {
		return "", fmt.Errorf("create container %q: %w", spec.Name, err)
	}
	return resp.ID, nil
}

func (c *Client) StartContainer(ctx context.Context, id string) error {
	return c.cli.ContainerStart(ctx, id, container.StartOptions{})
}

func (c *Client) StopContainer(ctx context.Context, id string, timeout time.Duration) error {
	t := int(timeout.Seconds())
	if timeout <= 0 {
		t = 10
	}
	return c.cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &t})
}

func (c *Client) RestartContainer(ctx context.Context, id string, timeout time.Duration) error {
	t := int(timeout.Seconds())
	if timeout <= 0 {
		t = 10
	}
	return c.cli.ContainerRestart(ctx, id, container.StopOptions{Timeout: &t})
}

func (c *Client) RemoveContainer(ctx context.Context, id string, force bool, removeVolumes bool) error {
	return c.cli.ContainerRemove(ctx, id, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: removeVolumes,
	})
}

func (c *Client) InspectContainer(ctx context.Context, id string) (*ContainerInfo, error) {
	ins, err := c.cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("inspect container %q: %w", id, err)
	}

	info := &ContainerInfo{
		ID:     ins.ID,
		Name:   strings.TrimPrefix(ins.Name, "/"),
		Image:  ins.Config.Image,
		State:  ins.State.Status,
		Status: ins.State.Status,
		Labels: ins.Config.Labels,
	}

	if ins.State != nil {
		if ins.State.ExitCode != 0 || ins.State.Status != "" {
			v := ins.State.ExitCode
			info.ExitCode = &v
		}
		if ins.State.StartedAt != "" {
			if t, err := time.Parse(time.RFC3339Nano, ins.State.StartedAt); err == nil {
				info.StartedAt = &t
			}
		}
		if ins.State.FinishedAt != "" {
			if t, err := time.Parse(time.RFC3339Nano, ins.State.FinishedAt); err == nil {
				info.FinishedAt = &t
			}
		}
		if ins.State.Health != nil {
			info.Health = ins.State.Health.Status
		}
	}

	return info, nil
}

func (c *Client) ListContainers(ctx context.Context, filter LabelFilter) ([]ContainerSummary, error) {
	args := filters.NewArgs()
	for k, v := range filter {
		args.Add("label", k+"="+v)
	}

	items, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: args,
	})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	out := make([]ContainerSummary, 0, len(items))
	for _, item := range items {
		summary := ContainerSummary{
			ID:     item.ID,
			Name:   strings.TrimPrefix(strings.Join(item.Names, ","), "/"),
			Image:  item.Image,
			State:  item.State,
			Status: item.Status,
			Labels: item.Labels,
		}
		if item.Created > 0 {
			summary.CreatedAt = time.Unix(item.Created, 0).UTC()
		}
		out = append(out, summary)
	}

	return out, nil
}

func (c *Client) ContainerLogs(ctx context.Context, id string, opts LogOptions) (io.ReadCloser, error) {
	logOpts := container.LogsOptions{
		ShowStdout: opts.Stdout,
		ShowStderr: opts.Stderr,
		Follow:     opts.Follow,
		Timestamps: opts.Timestamps,
		Tail:       opts.Tail,
	}
	if !opts.Since.IsZero() {
		logOpts.Since = opts.Since.UTC().Format(time.RFC3339)
	}

	rc, err := c.cli.ContainerLogs(ctx, id, logOpts)
	if err != nil {
		return nil, fmt.Errorf("container logs %q: %w", id, err)
	}
	return rc, nil
}

func (c *Client) WaitContainer(ctx context.Context, id string) (int64, error) {
	statusCh, errCh := c.cli.ContainerWait(ctx, id, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return 0, fmt.Errorf("wait container %q: %w", id, err)
		}
	case status := <-statusCh:
		return status.StatusCode, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	return 0, errors.New("container wait ended unexpectedly")
}

func (c *Client) Raw() *client.Client {
	return c.cli
}