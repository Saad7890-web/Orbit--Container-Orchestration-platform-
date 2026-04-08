package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Saad7890-web/orbit/internal/models"
)

var (
	// Practical Docker image validator for Orbit's first release.
	// This catches common invalid names while staying dependency-light.
	imageNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*(?::[A-Za-z0-9_][A-Za-z0-9_.-]{0,127})?(?:@sha256:[a-fA-F0-9]{64})?$`)
)

func Validate(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	if strings.TrimSpace(cfg.APIVersion) == "" {
		return errors.New("apiVersion is required")
	}
	if cfg.APIVersion != CurrentAPIVersion {
		return fmt.Errorf("unsupported apiVersion %q, expected %q", cfg.APIVersion, CurrentAPIVersion)
	}

	if strings.TrimSpace(cfg.Kind) == "" {
		return errors.New("kind is required")
	}
	if cfg.Kind != CurrentKind {
		return fmt.Errorf("unsupported kind %q, expected %q", cfg.Kind, CurrentKind)
	}

	if strings.TrimSpace(cfg.Metadata.Name) == "" {
		return errors.New("metadata.name is required")
	}

	if err := validateServices(cfg.Spec.Services); err != nil {
		return fmt.Errorf("services: %w", err)
	}
	if err := validateJobs(cfg.Spec.Jobs); err != nil {
		return fmt.Errorf("jobs: %w", err)
	}
	if err := validateTriggers(cfg.Spec.Triggers); err != nil {
		return fmt.Errorf("triggers: %w", err)
	}

	return nil
}

func validateServices(services []models.Service) error {
	seen := map[string]struct{}{}
	for i := range services {
		svc := services[i]
		if err := svc.Validate(); err != nil {
			return fmt.Errorf("[%d] %w", i, err)
		}
		if err := validateImageName(svc.Image); err != nil {
			return fmt.Errorf("[%s] image: %w", svc.Name, err)
		}
		if _, ok := seen[svc.Name]; ok {
			return fmt.Errorf("duplicate service name %q", svc.Name)
		}
		seen[svc.Name] = struct{}{}
	}
	return nil
}

func validateJobs(jobs []models.Job) error {
	seen := map[string]struct{}{}
	for i := range jobs {
		job := jobs[i]
		if err := job.Validate(); err != nil {
			return fmt.Errorf("[%d] %w", i, err)
		}
		if err := validateImageName(job.Image); err != nil {
			return fmt.Errorf("[%s] image: %w", job.Name, err)
		}
		if job.Schedule != nil {
			if err := job.Schedule.Validate(); err != nil {
				return fmt.Errorf("[%s] schedule: %w", job.Name, err)
			}
		}
		if _, ok := seen[job.Name]; ok {
			return fmt.Errorf("duplicate job name %q", job.Name)
		}
		seen[job.Name] = struct{}{}
	}
	return nil
}

func validateTriggers(triggers []models.Trigger) error {
	seen := map[string]struct{}{}
	for i := range triggers {
		trg := triggers[i]
		if err := trg.Validate(); err != nil {
			return fmt.Errorf("[%d] %w", i, err)
		}
		if _, ok := seen[trg.Name]; ok {
			return fmt.Errorf("duplicate trigger name %q", trg.Name)
		}
		seen[trg.Name] = struct{}{}
	}
	return nil
}

func validateImageName(image string) error {
	image = strings.TrimSpace(image)
	if image == "" {
		return errors.New("image is required")
	}
	if strings.ContainsAny(image, " \t\r\n") {
		return fmt.Errorf("invalid image name %q: must not contain whitespace", image)
	}
	if strings.HasPrefix(image, "/") || strings.HasSuffix(image, "/") {
		return fmt.Errorf("invalid image name %q: must not start or end with /", image)
	}
	if strings.Contains(image, "//") {
		return fmt.Errorf("invalid image name %q: contains empty path segment", image)
	}
	if !imageNamePattern.MatchString(image) {
		return fmt.Errorf("invalid image name %q", image)
	}
	return nil
}