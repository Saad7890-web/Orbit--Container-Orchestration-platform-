package daemon

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Saad7890-web/orbit/internal/events"
	"github.com/Saad7890-web/orbit/internal/models"
	"github.com/Saad7890-web/orbit/internal/triggers"
)

func (d *Daemon) Run(ctx context.Context) error {
	if d == nil {
		return fmt.Errorf("daemon is nil")
	}

	stack := d.stack()
	if err := stack.Validate(); err != nil {
		return err
	}

	if err := d.repo.SetMetadata(ctx, "daemon.config_path", d.opts.ConfigPath); err != nil {
		return err
	}
	if err := d.repo.SetMetadata(ctx, "daemon.db_path", d.opts.DBPath); err != nil {
		return err
	}
	if err := d.repo.SetMetadata(ctx, "daemon.http_addr", d.opts.HTTPAddr); err != nil {
		return err
	}
	if err := d.repo.SetMetadata(ctx, "daemon.config_hash:"+stack.Name, d.configHash); err != nil {
		return err
	}

	if d.bus != nil {
		_ = d.bus.PublishNow(events.Event{
			ID:        "daemon-started",
			Type:      events.TypeConfigApplied,
			Source:    "daemon",
			StackName: stack.Name,
			Timestamp: time.Now().UTC(),
			Data: map[string]any{
				"configHash": d.configHash,
			},
		})
	}

	report, err := d.controller.Apply(ctx, stack, d.configHash)
	if err != nil {
		return err
	}

	if d.bus != nil {
		_ = d.bus.PublishNow(events.Event{
			ID:        "stack-reconciled",
			Type:      events.TypeStackReconciled,
			Source:    "controller",
			StackName: stack.Name,
			Timestamp: time.Now().UTC(),
			Data: map[string]any{
				"configHash": d.configHash,
				"created":     report.Created,
				"updated":     report.Updated,
				"recreated":   report.Recreated,
				"removed":     report.Removed,
				"skipped":     report.Skipped,
			},
		})
	}

	if err := d.sched.Sync(stack.Name, stack.Jobs, d.configHash); err != nil {
		return err
	}
	if err := d.triggerEng.Sync(stack.Name, d.configHash, stack.Triggers); err != nil {
		return err
	}

	var wg sync.WaitGroup

	// Start the trigger HTTP server.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := d.httpServer.Start(); err != nil && ctx.Err() == nil {
			_ = d.bus.PublishNow(events.Event{
				ID:        "trigger-http-error",
				Type:      events.TypeError,
				Source:    "http-server",
				StackName: stack.Name,
				Timestamp: time.Now().UTC(),
				Data: map[string]any{
					"error": err.Error(),
				},
			})
		}
	}()

	// Start the scheduler loop.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := d.sched.Start(ctx); err != nil && err != context.Canceled {
			_ = d.bus.PublishNow(events.Event{
				ID:        "scheduler-error",
				Type:      events.TypeError,
				Source:    "scheduler",
				StackName: stack.Name,
				Timestamp: time.Now().UTC(),
				Data: map[string]any{
					"error": err.Error(),
				},
			})
		}
	}()

	// Start service monitors.
	for _, svc := range stack.Services {
		svc := svc
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = d.lifecycle.Watch(ctx, stack.Name, svc, d.configHash)
		}()
	}

	// Block until shutdown.
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = d.httpServer.Shutdown(shutdownCtx)
	_ = d.triggerEng.Sync(stack.Name, d.configHash, nil)

	_ = d.runtime.Close()
	_ = d.store.Close()
	_ = d.bus.Close()

	wg.Wait()
	return ctx.Err()
}

func (d *Daemon) dispatchTrigger(ctx context.Context, inv triggers.Invocation) error {
	stack := d.stack()

	switch inv.Target.Kind {
	case models.WorkloadKindJob:
		for _, job := range stack.Jobs {
			if job.Name != inv.Target.Name {
				continue
			}
			_, err := d.jobRunner.RunOnce(ctx, stack.Name, job, d.configHash)
			return err
		}
		return fmt.Errorf("job %q not found", inv.Target.Name)

	case models.WorkloadKindService:
		return fmt.Errorf("service trigger targets are not wired yet")

	default:
		return fmt.Errorf("unsupported trigger target kind %q", inv.Target.Kind)
	}
}