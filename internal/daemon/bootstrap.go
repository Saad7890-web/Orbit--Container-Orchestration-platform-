package daemon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/Saad7890-web/orbit/internal/config"
	"github.com/Saad7890-web/orbit/internal/controller"
	orbitdocker "github.com/Saad7890-web/orbit/internal/docker"
	"github.com/Saad7890-web/orbit/internal/events"
	"github.com/Saad7890-web/orbit/internal/models"
	"github.com/Saad7890-web/orbit/internal/scheduler"
	"github.com/Saad7890-web/orbit/internal/state"
	"github.com/Saad7890-web/orbit/internal/triggers"
)

type Options struct {
	ConfigPath   string
	DBPath       string
	HTTPAddr     string
	DockerHost   string
	UseDockerEnv bool
}

type Daemon struct {
	opts         Options
	cfg          *config.Config
	configHash   string
	store        *state.Store
	repo         *state.Repository
	runtime      orbitdocker.Runtime
	controller   *controller.Controller
	jobRunner    *scheduler.JobRunner
	sched        *scheduler.Scheduler
	triggerEng   *triggers.Engine
	httpServer   *triggers.HTTPServer
	lifecycle    *controller.ServiceLifecycleManager
	bus          *events.Bus
}

func Bootstrap(ctx context.Context, opts Options) (*Daemon, error) {
	if opts.ConfigPath == "" {
		opts.ConfigPath = "orbit.yaml"
	}
	if opts.DBPath == "" {
		opts.DBPath = "orbit.db"
	}
	if opts.HTTPAddr == "" {
		opts.HTTPAddr = "127.0.0.1:8081"
	}

	raw, err := os.ReadFile(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", opts.ConfigPath, err)
	}

	cfg, err := config.Parse(raw)
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256(raw)
	configHash := "sha256:" + hex.EncodeToString(sum[:])

	store, err := state.Open(ctx, opts.DBPath)
	if err != nil {
		return nil, err
	}

	runtime, err := orbitdocker.New(ctx, orbitdocker.Options{
		Host:        opts.DockerHost,
		UseEnv:      opts.UseDockerEnv,
		APIVersion:  true,
		PingTimeout: 5 * time.Second,
	})
	if err != nil {
		_ = store.Close()
		return nil, err
	}

	repo := store.Repository()
	ctrl := controller.New(repo, runtime)

	bus := events.NewBus()

	da := &Daemon{
		opts:       opts,
		cfg:        cfg,
		configHash: configHash,
		store:      store,
		repo:       repo,
		runtime:    runtime,
		controller: ctrl,
		bus:        bus,
	}

	da.jobRunner = scheduler.NewJobRunner(runtime, repo)
	da.sched = scheduler.New(da.jobRunner)
	da.lifecycle = controller.NewServiceLifecycleManager(runtime, repo, 5*time.Second)
	da.triggerEng = triggers.NewEngine(da.dispatchTrigger)
	da.httpServer = triggers.NewHTTPServer(opts.HTTPAddr, da.triggerEng)

	return da, nil
}

func (d *Daemon) stack() models.Stack {
	if d == nil || d.cfg == nil {
		return models.Stack{}
	}

	return models.Stack{
		Name:        d.cfg.Metadata.Name,
		Description: d.cfg.Metadata.Description,
		Version:     d.cfg.APIVersion,
		Labels:      d.cfg.Metadata.Labels,
		Services:    d.cfg.Spec.Services,
		Jobs:        d.cfg.Spec.Jobs,
		Triggers:    d.cfg.Spec.Triggers,
	}
}