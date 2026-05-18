package events

import (
	"context"
	"fmt"
	"sync"
)

type Handler interface {
	Handle(ctx context.Context, e Event) error
}

type HandlerFuncAdapter func(ctx context.Context, e Event) error

func (f HandlerFuncAdapter) Handle(ctx context.Context, e Event) error {
	return f(ctx, e)
}

type Router struct {
	mu       sync.RWMutex
	handlers map[Type][]Handler
	fallback []Handler
}

func NewRouter() *Router {
	return &Router{
		handlers: make(map[Type][]Handler),
	}
}

func (r *Router) Register(eventType Type, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[eventType] = append(r.handlers[eventType], h)
}

func (r *Router) RegisterFunc(eventType Type, fn func(context.Context, Event) error) {
	r.Register(eventType, HandlerFuncAdapter(fn))
}

func (r *Router) RegisterFallback(h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallback = append(r.fallback, h)
}

func (r *Router) Dispatch(ctx context.Context, e Event) error {
	r.mu.RLock()
	handlers := append([]Handler(nil), r.handlers[e.Type]...)
	fallback := append([]Handler(nil), r.fallback...)
	r.mu.RUnlock()

	if len(handlers) == 0 {
		handlers = fallback
	}

	if len(handlers) == 0 {
		return nil
	}

	var errs []error
	for _, h := range handlers {
		if h == nil {
			continue
		}
		if err := h.Handle(ctx, e); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return joinErrors(errs...)
}

func joinErrors(errs ...error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	msg := "multiple handler errors:"
	for _, err := range errs {
		msg += " " + err.Error() + ";"
	}
	return fmt.Errorf("%s", msg)
}