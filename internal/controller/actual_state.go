package controller

import (
	"context"
	"fmt"

	"github.com/Saad7890-web/orbit/internal/docker"
)

type ActualState struct {
	Services map[string]docker.ContainerSummary
}

func LoadActualState(ctx context.Context, runtime docker.Runtime, stackName string) (*ActualState, error) {
	if runtime == nil {
		return nil, fmt.Errorf("docker runtime is nil")
	}

	items, err := runtime.ListContainers(ctx, docker.LabelFilter{
		docker.LabelManaged: "true",
		docker.LabelStack:   stackName,
	})
	if err != nil {
		return nil, err
	}

	out := &ActualState{
		Services: make(map[string]docker.ContainerSummary, len(items)),
	}

	for _, item := range items {
		workloadName := item.Labels[docker.LabelWorkload]
		if workloadName == "" {
			continue
		}
		out.Services[workloadName] = item
	}

	return out, nil
}

func (a *ActualState) Service(name string) (docker.ContainerSummary, bool) {
	if a == nil {
		return docker.ContainerSummary{}, false
	}
	s, ok := a.Services[name]
	return s, ok
}