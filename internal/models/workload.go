package models

import (
	"errors"
	"fmt"
	"strings"
)

type Stack struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string            `json:"version,omitempty" yaml:"version,omitempty"`
	Services    []Service         `json:"services,omitempty" yaml:"services,omitempty"`
	Jobs        []Job             `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	Triggers    []Trigger         `json:"triggers,omitempty" yaml:"triggers,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

func (s Stack) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("stack name is required")
	}
	for i := range s.Services {
		if err := s.Services[i].Validate(); err != nil {
			return fmt.Errorf("service[%d]: %w", i, err)
		}
	}
	for i := range s.Jobs {
		if err := s.Jobs[i].Validate(); err != nil {
			return fmt.Errorf("job[%d]: %w", i, err)
		}
	}
	for i := range s.Triggers {
		if err := s.Triggers[i].Validate(); err != nil {
			return fmt.Errorf("trigger[%d]: %w", i, err)
		}
	}
	return nil
}

type WorkloadRef struct {
	Kind WorkloadKind `json:"kind" yaml:"kind"`
	Name string       `json:"name" yaml:"name"`
}

func (r WorkloadRef) Validate() error {
	if !r.Kind.Valid() {
		return ErrInvalidEnum("workload kind", string(r.Kind))
	}
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("workload name is required")
	}
	return nil
}