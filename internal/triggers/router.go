package triggers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Saad7890-web/orbit/internal/models"
)

type Payload struct {
	Source   string            `json:"source"`
	Event    string            `json:"event"`
	Path     string            `json:"path,omitempty"`
	Method   string            `json:"method,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     []byte            `json:"body,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type Invocation struct {
	StackName   string        `json:"stackName"`
	TriggerName string        `json:"triggerName"`
	TriggerType models.TriggerType `json:"triggerType"`
	Target      models.TriggerTarget `json:"target"`
	Payload     Payload       `json:"payload"`
}

type Dispatcher func(ctx context.Context, inv Invocation) error

func matchHTTPTrigger(t models.Trigger, path, method string) bool {
	if t.Type != models.TriggerHTTP || t.Match.HTTP == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(t.Match.HTTP.Path), strings.TrimSpace(path)) {
		return false
	}
	want := strings.ToUpper(strings.TrimSpace(t.Match.HTTP.Method))
	got := strings.ToUpper(strings.TrimSpace(method))
	if want == "" {
		want = http.MethodPost
	}
	if got == "" {
		got = http.MethodPost
	}
	return want == got
}

func matchFileTrigger(t models.Trigger, path, eventType string) bool {
	if t.Type != models.TriggerFile || t.Match.File == nil {
		return false
	}
	if strings.TrimSpace(t.Match.File.Path) != strings.TrimSpace(path) {
		return false
	}
	want := strings.ToLower(strings.TrimSpace(t.Match.File.EventType))
	got := strings.ToLower(strings.TrimSpace(eventType))
	if want == "" {
		want = "create"
	}
	if got == "" {
		got = "write"
	}
	return want == got
}

func pathToTriggerName(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/")
	path = strings.ReplaceAll(path, "/", ".")
	if path == "" {
		return "root"
	}
	return path
}

func validateTriggerSet(triggers []models.Trigger) error {
	seen := map[string]struct{}{}
	for i := range triggers {
		t := triggers[i]
		if err := t.Validate(); err != nil {
			return fmt.Errorf("[%d] %w", i, err)
		}
		if _, ok := seen[t.Name]; ok {
			return fmt.Errorf("duplicate trigger name %q", t.Name)
		}
		seen[t.Name] = struct{}{}
	}
	return nil
}