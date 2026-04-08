package config

import "github.com/Saad7890-web/orbit/internal/models"

const (
	CurrentAPIVersion = "orbit.dev/v1alpha1"
	CurrentKind       = "Stack"
)

type Metadata struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type Spec struct {
	Services []models.Service `json:"services,omitempty" yaml:"services,omitempty"`
	Jobs     []models.Job     `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	Triggers []models.Trigger `json:"triggers,omitempty" yaml:"triggers,omitempty"`
}

type Config struct {
	APIVersion string   `json:"apiVersion" yaml:"apiVersion"`
	Kind       string   `json:"kind" yaml:"kind"`
	Metadata   Metadata `json:"metadata" yaml:"metadata"`
	Spec       Spec     `json:"spec" yaml:"spec"`
}