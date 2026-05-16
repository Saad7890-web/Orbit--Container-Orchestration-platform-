package triggers

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/Saad7890-web/orbit/internal/models"
)

type Engine struct {
	mu         sync.RWMutex
	stackName  string
	configHash string
	triggers   map[string]models.Trigger
	dispatch   Dispatcher
}

func NewEngine(dispatch Dispatcher) *Engine {
	return &Engine{
		triggers: make(map[string]models.Trigger),
		dispatch: dispatch,
	}
}

func (e *Engine) Sync(stackName string, configHash string, triggers []models.Trigger) error {
	if e == nil {
		return fmt.Errorf("engine is nil")
	}
	if err := validateTriggerSet(triggers); err != nil {
		return err
	}

	next := make(map[string]models.Trigger, len(triggers))
	for _, t := range triggers {
		next[t.Name] = t
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.stackName = stackName
	e.configHash = configHash
	e.triggers = next
	return nil
}

func (e *Engine) List() []models.Trigger {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	out := make([]models.Trigger, 0, len(e.triggers))
	for _, t := range e.triggers {
		out = append(out, t)
	}
	return out
}

func (e *Engine) HandleHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if e == nil {
		http.Error(w, "trigger engine unavailable", http.StatusServiceUnavailable)
		return
	}

	e.mu.RLock()
	stackName := e.stackName
	configHash := e.configHash
	triggers := make([]models.Trigger, 0, len(e.triggers))
	for _, t := range e.triggers {
		triggers = append(triggers, t)
	}
	e.mu.RUnlock()

	for _, t := range triggers {
		if !matchHTTPTrigger(t, r.URL.Path, r.Method) {
			continue
		}

		body := make([]byte, 0)
		if r.Body != nil {
			defer r.Body.Close()
			buf := make([]byte, 32*1024)
			for {
				n, err := r.Body.Read(buf)
				if n > 0 {
					body = append(body, buf[:n]...)
				}
				if err != nil {
					break
				}
			}
		}

		headers := map[string]string{}
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		inv := Invocation{
			StackName:   stackName,
			TriggerName: t.Name,
			TriggerType: t.Type,
			Target:      t.Target,
			Payload: Payload{
				Source:   "http",
				Event:    "webhook",
				Path:     r.URL.Path,
				Method:   r.Method,
				Headers:  headers,
				Body:     body,
				Metadata: map[string]string{"configHash": configHash},
			},
		}

		if e.dispatch != nil {
			if err := e.dispatch(ctx, inv); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"accepted","trigger":"` + t.Name + `"}`))
		return
	}

	http.Error(w, "no matching trigger", http.StatusNotFound)
}

func (e *Engine) HandleFileEvent(ctx context.Context, path, eventType string, metadata map[string]string) error {
	if e == nil {
		return fmt.Errorf("trigger engine is nil")
	}

	e.mu.RLock()
	stackName := e.stackName
	configHash := e.configHash
	triggers := make([]models.Trigger, 0, len(e.triggers))
	for _, t := range e.triggers {
		triggers = append(triggers, t)
	}
	e.mu.RUnlock()

	for _, t := range triggers {
		if !matchFileTrigger(t, path, eventType) {
			continue
		}

		inv := Invocation{
			StackName:   stackName,
			TriggerName: t.Name,
			TriggerType: t.Type,
			Target:      t.Target,
			Payload: Payload{
				Source:   "file",
				Event:    eventType,
				Path:     path,
				Metadata: mergeMeta(metadata, map[string]string{"configHash": configHash}),
			},
		}

		if e.dispatch != nil {
			return e.dispatch(ctx, inv)
		}
		return nil
	}

	return nil
}

func mergeMeta(a, b map[string]string) map[string]string {
	out := make(map[string]string, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}