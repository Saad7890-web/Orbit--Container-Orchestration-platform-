package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/Saad7890-web/orbit/internal/docker"
	"github.com/Saad7890-web/orbit/internal/models"
)

type Report struct {
	StackName string   `json:"stackName"`
	Created   []string `json:"created,omitempty"`
	Updated   []string `json:"updated,omitempty"`
	Recreated []string `json:"recreated,omitempty"`
	Removed   []string `json:"removed,omitempty"`
	Skipped   []string `json:"skipped,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

type Reconciler struct {
	repo   Repository
	runtime docker.Runtime
}

func NewReconciler(repo Repository, runtime docker.Runtime) *Reconciler {
	return &Reconciler{
		repo:   repo,
		runtime: runtime,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, desired DesiredState) (*Report, error) {
	if r == nil {
		return nil, fmt.Errorf("reconciler is nil")
	}
	if r.repo == nil {
		return nil, fmt.Errorf("repository is nil")
	}
	if r.runtime == nil {
		return nil, fmt.Errorf("docker runtime is nil")
	}

	if err := desired.Stack.Validate(); err != nil {
		return nil, err
	}

	report := &Report{StackName: desired.Stack.Name}

	if err := r.repo.SaveStack(ctx, desired.Stack, desired.ConfigHash); err != nil {
		return nil, err
	}

	if err := r.repo.SetMetadata(ctx, "last_applied_stack", desired.Stack.Name); err != nil {
		return nil, err
	}
	if err := r.repo.SetMetadata(ctx, "last_applied_config_hash:"+desired.Stack.Name, desired.ConfigHash); err != nil {
		return nil, err
	}

	desiredServices := map[string]models.Service{}
	for _, svc := range desired.Stack.Services {
		desiredServices[svc.Name] = svc
	}

	desiredJobs := map[string]models.Job{}
	for _, job := range desired.Stack.Jobs {
		desiredJobs[job.Name] = job
	}

	desiredTriggers := map[string]models.Trigger{}
	for _, trg := range desired.Stack.Triggers {
		desiredTriggers[trg.Name] = trg
	}

	// Persist jobs and triggers first.
	for _, job := range desired.Stack.Jobs {
		if err := r.repo.UpsertJob(ctx, desired.Stack.Name, job, desired.ConfigHash, models.ExecutionPending, "", nil); err != nil {
			return nil, fmt.Errorf("persist job %q: %w", job.Name, err)
		}
	}
	for _, trg := range desired.Stack.Triggers {
		if err := r.repo.UpsertTrigger(ctx, desired.Stack.Name, trg, desired.ConfigHash, models.ExecutionPending, "", nil); err != nil {
			return nil, fmt.Errorf("persist trigger %q: %w", trg.Name, err)
		}
	}

	actual, err := LoadActualState(ctx, r.runtime, desired.Stack.Name)
	if err != nil {
		return nil, err
	}

	// Reconcile services.
	for _, svc := range desired.Stack.Services {
		if !svc.Enabled && svc.Enabled == false {
			// Enabled defaults to false in zero value, so treat explicitly disabled only
			// if the field is present. Orbit should still manage normal services later.
			// For now, zero-value false does not mean disabled.
		}

		containerName := docker.BuildContainerName(desired.Stack.Name, svc.Name)
		containerID, exists := actual.Service(svc.Name)

		if !exists {
			if err := r.createAndStartService(ctx, desired.Stack.Name, svc, desired.ConfigHash, containerName, report); err != nil {
				return nil, err
			}
			continue
		}

		info, err := r.runtime.InspectContainer(ctx, containerID.ID)
		if err != nil {
			if err := r.createAndStartService(ctx, desired.Stack.Name, svc, desired.ConfigHash, containerName, report); err != nil {
				return nil, err
			}
			continue
		}

		currentHash := ""
		if info != nil && info.Labels != nil {
			currentHash = info.Labels[docker.LabelHash]
		}

		needsReplace := currentHash != desired.ConfigHash || info.Image != svc.Image
		if needsReplace {
			if err := r.runtime.StopContainer(ctx, containerID.ID, 10*time.Second); err != nil {
				report.Warnings = append(report.Warnings, fmt.Sprintf("stop %s: %v", svc.Name, err))
			}
			if err := r.runtime.RemoveContainer(ctx, containerID.ID, true, true); err != nil {
				return nil, fmt.Errorf("remove old service %q: %w", svc.Name, err)
			}
			if err := r.createAndStartService(ctx, desired.Stack.Name, svc, desired.ConfigHash, containerName, report); err != nil {
				return nil, err
			}
			report.Recreated = append(report.Recreated, svc.Name)
			continue
		}

		if info.State != "running" {
			if err := r.runtime.StartContainer(ctx, containerID.ID); err != nil {
				return nil, fmt.Errorf("start service %q: %w", svc.Name, err)
			}
			report.Updated = append(report.Updated, svc.Name)
		} else {
			report.Skipped = append(report.Skipped, svc.Name)
		}

		health := models.HealthUnknown
		if info.Health == "healthy" {
			health = models.HealthHealthy
		} else if info.Health == "unhealthy" {
			health = models.HealthUnhealthy
		} else if info.Health == "starting" {
			health = models.HealthDegraded
		}

		if err := r.repo.UpsertService(
			ctx,
			desired.Stack.Name,
			svc,
			desired.ConfigHash,
			models.LifecycleRunning,
			health,
			"",
		); err != nil {
			return nil, fmt.Errorf("persist service %q: %w", svc.Name, err)
		}
	}

	// Prune removed managed services.
	for workloadName, summary := range actual.Services {
		if _, ok := desiredServices[workloadName]; ok {
			continue
		}
		if summary.Labels[docker.LabelManaged] != "true" {
			continue
		}
		if err := r.runtime.StopContainer(ctx, summary.ID, 10*time.Second); err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("stop removed %s: %v", workloadName, err))
		}
		if err := r.runtime.RemoveContainer(ctx, summary.ID, true, true); err != nil {
			return nil, fmt.Errorf("remove stale service %q: %w", workloadName, err)
		}
		report.Removed = append(report.Removed, workloadName)
	}

	// Persist jobs/triggers again so state reflects the latest config hash.
	for _, job := range desired.Stack.Jobs {
		if err := r.repo.UpsertJob(ctx, desired.Stack.Name, job, desired.ConfigHash, models.ExecutionPending, "", nil); err != nil {
			return nil, fmt.Errorf("persist job %q: %w", job.Name, err)
		}
	}
	for _, trg := range desired.Stack.Triggers {
		if err := r.repo.UpsertTrigger(ctx, desired.Stack.Name, trg, desired.ConfigHash, models.ExecutionPending, "", nil); err != nil {
			return nil, fmt.Errorf("persist trigger %q: %w", trg.Name, err)
		}
	}

	return report, nil
}

func (r *Reconciler) createAndStartService(
	ctx context.Context,
	stackName string,
	svc models.Service,
	configHash string,
	containerName string,
	report *Report,
) error {
	spec := docker.ContainerSpec{
		Name:        containerName,
		StackName:   stackName,
		Kind:        models.WorkloadKindService,
		Workload:    svc.Name,
		Image:       svc.Image,
		Command:     svc.Command,
		WorkingDir:  svc.WorkingDir,
		Env:         svc.Env,
		Labels:      svc.Labels,
		Ports:       svc.Ports,
		Volumes:     svc.Volumes,
		Restart:     svc.Restart,
		HealthCheck: svc.HealthCheck,
	}

	id, err := r.runtime.CreateContainer(ctx, spec)
	if err != nil {
		return fmt.Errorf("create service %q: %w", svc.Name, err)
	}
	if err := r.runtime.StartContainer(ctx, id); err != nil {
		return fmt.Errorf("start service %q: %w", svc.Name, err)
	}

	health := models.HealthUnknown
	if err := r.repo.UpsertService(
		ctx,
		stackName,
		svc,
		configHash,
		models.LifecycleRunning,
		health,
		"",
	); err != nil {
		return fmt.Errorf("persist service %q: %w", svc.Name, err)
	}

	report.Created = append(report.Created, svc.Name)
	return nil
}