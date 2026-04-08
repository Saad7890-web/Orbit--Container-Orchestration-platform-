package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Schedule struct {
	Cron     string `json:"cron" yaml:"cron"`
	Timezone string `json:"timezone,omitempty" yaml:"timezone,omitempty"`
}

func (s Schedule) Validate() error {
	if strings.TrimSpace(s.Cron) == "" {
		return errors.New("cron schedule is required")
	}
	// Lightweight validation here; full cron parsing can come in scheduler package later.
	fields := strings.Fields(s.Cron)
	if len(fields) != 5 && len(fields) != 6 {
		return fmt.Errorf("invalid cron expression %q", s.Cron)
	}
	return nil
}

type Job struct {
	Name        string            `json:"name" yaml:"name"`
	Image       string            `json:"image" yaml:"image"`
	Schedule    *Schedule         `json:"schedule,omitempty" yaml:"schedule,omitempty"`
	Command     []string          `json:"command,omitempty" yaml:"command,omitempty"`
	Args        []string          `json:"args,omitempty" yaml:"args,omitempty"`
	WorkingDir  string            `json:"workingDir,omitempty" yaml:"workingDir,omitempty"`
	Env         map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Volumes     []VolumeMount     `json:"volumes,omitempty" yaml:"volumes,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Timeout     string            `json:"timeout,omitempty" yaml:"timeout,omitempty"` // e.g. "30m"
	RetryCount  int               `json:"retryCount,omitempty" yaml:"retryCount,omitempty"`
	Enabled     bool              `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

var jobNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

func (j Job) Validate() error {
	if strings.TrimSpace(j.Name) == "" {
		return errors.New("job name is required")
	}
	if !jobNamePattern.MatchString(j.Name) {
		return fmt.Errorf("invalid job name %q", j.Name)
	}
	if strings.TrimSpace(j.Image) == "" {
		return errors.New("job image is required")
	}
	if j.Schedule != nil {
		if err := j.Schedule.Validate(); err != nil {
			return fmt.Errorf("schedule: %w", err)
		}
	}
	if j.Timeout != "" {
		if _, err := time.ParseDuration(j.Timeout); err != nil {
			return fmt.Errorf("invalid timeout %q: %w", j.Timeout, err)
		}
	}
	if j.RetryCount < 0 {
		return errors.New("retryCount must be >= 0")
	}
	for i := range j.Volumes {
		if err := j.Volumes[i].Validate(); err != nil {
			return fmt.Errorf("volumes[%d]: %w", i, err)
		}
	}
	return nil
}