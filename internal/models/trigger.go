package models

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

type TriggerType string

const (
	TriggerHTTP  TriggerType = "http"
	TriggerMQTT  TriggerType = "mqtt"
	TriggerFile  TriggerType = "file"
	TriggerEvent TriggerType = "event"
)

func (t TriggerType) Valid() bool {
	switch t {
	case TriggerHTTP, TriggerMQTT, TriggerFile, TriggerEvent:
		return true
	default:
		return false
	}
}

type HTTPTriggerMatch struct {
	Path    string            `json:"path" yaml:"path"`
	Method  string            `json:"method,omitempty" yaml:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

func (m HTTPTriggerMatch) Validate() error {
	if strings.TrimSpace(m.Path) == "" {
		return errors.New("http trigger path is required")
	}
	if !strings.HasPrefix(m.Path, "/") {
		return fmt.Errorf("http trigger path must start with /: %q", m.Path)
	}
	switch strings.ToUpper(m.Method) {
	case "", "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return nil
	default:
		return fmt.Errorf("invalid http method %q", m.Method)
	}
}

type MQTTTriggerMatch struct {
	Topic   string `json:"topic" yaml:"topic"`
	Payload string `json:"payload,omitempty" yaml:"payload,omitempty"`
}

func (m MQTTTriggerMatch) Validate() error {
	if strings.TrimSpace(m.Topic) == "" {
		return errors.New("mqtt trigger topic is required")
	}
	return nil
}

type FileTriggerMatch struct {
	Path      string `json:"path" yaml:"path"`
	EventType string `json:"eventType,omitempty" yaml:"eventType,omitempty"` // create, write, remove, rename
}

func (m FileTriggerMatch) Validate() error {
	if strings.TrimSpace(m.Path) == "" {
		return errors.New("file trigger path is required")
	}
	if !strings.HasPrefix(m.Path, "/") {
		return fmt.Errorf("file trigger path must be absolute: %q", m.Path)
	}
	switch m.EventType {
	case "", "create", "write", "remove", "rename":
		return nil
	default:
		return fmt.Errorf("invalid file eventType %q", m.EventType)
	}
}

type TriggerMatch struct {
	HTTP  *HTTPTriggerMatch  `json:"http,omitempty" yaml:"http,omitempty"`
	MQTT  *MQTTTriggerMatch  `json:"mqtt,omitempty" yaml:"mqtt,omitempty"`
	File  *FileTriggerMatch  `json:"file,omitempty" yaml:"file,omitempty"`
}

func (m TriggerMatch) Validate() error {
	set := 0
	if m.HTTP != nil {
		set++
		if err := m.HTTP.Validate(); err != nil {
			return fmt.Errorf("http: %w", err)
		}
	}
	if m.MQTT != nil {
		set++
		if err := m.MQTT.Validate(); err != nil {
			return fmt.Errorf("mqtt: %w", err)
		}
	}
	if m.File != nil {
		set++
		if err := m.File.Validate(); err != nil {
			return fmt.Errorf("file: %w", err)
		}
	}
	if set == 0 {
		return errors.New("at least one trigger match rule is required")
	}
	if set > 1 {
		return errors.New("only one trigger match rule is allowed per trigger")
	}
	return nil
}

type TriggerTarget struct {
	Kind WorkloadKind `json:"kind" yaml:"kind"`
	Name string       `json:"name" yaml:"name"`
}

func (t TriggerTarget) Validate() error {
	return WorkloadRef{Kind: t.Kind, Name: t.Name}.Validate()
}

type Trigger struct {
	Name        string        `json:"name" yaml:"name"`
	Type        TriggerType   `json:"type" yaml:"type"`
	Match       TriggerMatch  `json:"match" yaml:"match"`
	Target      TriggerTarget `json:"target" yaml:"target"`
	Enabled     bool          `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Description string        `json:"description,omitempty" yaml:"description,omitempty"`
}

func (t Trigger) Validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return errors.New("trigger name is required")
	}
	if !t.Type.Valid() {
		return ErrInvalidEnum("trigger type", string(t.Type))
	}
	if err := t.Match.Validate(); err != nil {
		return fmt.Errorf("match: %w", err)
	}
	if err := t.Target.Validate(); err != nil {
		return fmt.Errorf("target: %w", err)
	}

	// Cross-check type and match shape for clarity.
	switch t.Type {
	case TriggerHTTP:
		if t.Match.HTTP == nil {
			return errors.New("http trigger requires http match rules")
		}
	case TriggerMQTT:
		if t.Match.MQTT == nil {
			return errors.New("mqtt trigger requires mqtt match rules")
		}
	case TriggerFile:
		if t.Match.File == nil {
			return errors.New("file trigger requires file match rules")
		}
	case TriggerEvent:
		// Generic internal event trigger; match rules are still required.
	}

	return nil
}

func IsValidHostPort(hostPort string) bool {
	_, _, err := net.SplitHostPort(hostPort)
	return err == nil
}

func IsValidURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme != "" && u.Host != ""
}