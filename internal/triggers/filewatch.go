package triggers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type FileWatcher struct {
	engine *Engine
	w      *fsnotify.Watcher

	mu      sync.RWMutex
	routes  map[string]string // exact watched path -> trigger path
	closed  bool
}

func NewFileWatcher(engine *Engine) (*FileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create file watcher: %w", err)
	}

	return &FileWatcher{
		engine: engine,
		w:      w,
		routes: make(map[string]string),
	}, nil
}

func (f *FileWatcher) Close() error {
	if f == nil || f.w == nil {
		return nil
	}
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
	return f.w.Close()
}

func (f *FileWatcher) Sync(paths []string) error {
	if f == nil || f.w == nil {
		return fmt.Errorf("file watcher is nil")
	}

	for _, p := range paths {
		p = filepath.Clean(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		dir := p
		if stat, err := os.Stat(p); err == nil && stat.IsDir() {
			dir = p
		} else {
			dir = filepath.Dir(p)
		}
		if err := f.w.Add(dir); err != nil {
			return fmt.Errorf("watch %q: %w", dir, err)
		}
		f.mu.Lock()
		f.routes[p] = p
		f.mu.Unlock()
	}

	return nil
}

func (f *FileWatcher) Start(ctx context.Context) error {
	if f == nil || f.w == nil {
		return fmt.Errorf("file watcher is nil")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case err, ok := <-f.w.Errors:
			if !ok {
				return nil
			}
			_ = err // keep running; errors are non-fatal in watch loops

		case ev, ok := <-f.w.Events:
			if !ok {
				return nil
			}
			if err := f.handleEvent(ctx, ev); err != nil {
				// keep watcher alive; trigger errors should not kill the loop
				continue
			}
		}
	}
}

func (f *FileWatcher) handleEvent(ctx context.Context, ev fsnotify.Event) error {
	if f == nil || f.engine == nil {
		return nil
	}

	path := filepath.Clean(ev.Name)
	eventType := fileEventType(ev)

	if eventType == "" {
		return nil
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// Direct path match.
	if _, ok := f.routes[path]; ok {
		return f.engine.HandleFileEvent(ctx, path, eventType, map[string]string{
			"op": ev.Op.String(),
		})
	}

	// Exact trigger target path match.
	return f.engine.HandleFileEvent(ctx, path, eventType, map[string]string{
		"op": ev.Op.String(),
	})
}

func fileEventType(ev fsnotify.Event) string {
	switch {
	case ev.Op&fsnotify.Create == fsnotify.Create:
		return "create"
	case ev.Op&fsnotify.Write == fsnotify.Write:
		return "write"
	case ev.Op&fsnotify.Remove == fsnotify.Remove:
		return "remove"
	case ev.Op&fsnotify.Rename == fsnotify.Rename:
		return "rename"
	default:
		return ""
	}
}