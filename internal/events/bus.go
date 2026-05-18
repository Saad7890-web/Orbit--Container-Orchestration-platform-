package events

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrBusClosed      = errors.New("event bus closed")
	ErrInvalidEvent   = errors.New("invalid event")
	ErrSubscriptionMissing = errors.New("subscription missing")
)

type Bus struct {
	mu          sync.RWMutex
	closed      bool
	nextSubID   atomic.Int64
	subscribers map[int]subscription
	wildcards   map[int]subscription
}

type subscription struct {
	ch     chan Event
	types  map[Type]struct{} // nil means wildcard
	closed bool
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[int]subscription),
		wildcards:   make(map[int]subscription),
	}
}

func (b *Bus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	for id, sub := range b.subscribers {
		if !sub.closed {
			close(sub.ch)
			sub.closed = true
		}
		delete(b.subscribers, id)
	}
	for id, sub := range b.wildcards {
		if !sub.closed {
			close(sub.ch)
			sub.closed = true
		}
		delete(b.wildcards, id)
	}

	return nil
}

func (b *Bus) Subscribe(buffer int, types ...Type) (Subscription, error) {
	if buffer <= 0 {
		buffer = 32
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return Subscription{}, ErrBusClosed
	}

	id := int(b.nextSubID.Add(1))
	ch := make(chan Event, buffer)

	sub := subscription{
		ch:    ch,
		types: nil,
	}

	if len(types) > 0 {
		sub.types = make(map[Type]struct{}, len(types))
		for _, t := range types {
			sub.types[t] = struct{}{}
		}
	}

	if sub.types == nil {
		b.wildcards[id] = sub
	} else {
		b.subscribers[id] = sub
	}

	return Subscription{
		ID:     id,
		Events: ch,
	}, nil
}

func (b *Bus) Unsubscribe(id int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return ErrBusClosed
	}

	if sub, ok := b.subscribers[id]; ok {
		if !sub.closed {
			close(sub.ch)
			sub.closed = true
		}
		delete(b.subscribers, id)
		return nil
	}

	if sub, ok := b.wildcards[id]; ok {
		if !sub.closed {
			close(sub.ch)
			sub.closed = true
		}
		delete(b.wildcards, id)
		return nil
	}

	return ErrSubscriptionMissing
}

func (b *Bus) Publish(ctx context.Context, e Event) error {
	if err := e.Validate(); err != nil {
		return err
	}

	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrBusClosed
	}

	subs := make([]subscription, 0, len(b.subscribers)+len(b.wildcards))
	for _, sub := range b.wildcards {
		subs = append(subs, sub)
	}
	for _, sub := range b.subscribers {
		if sub.matches(e.Type) {
			subs = append(subs, sub)
		}
	}
	b.mu.RUnlock()

	for _, sub := range subs {
		select {
		case sub.ch <- e:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Do not block the whole system because one consumer is slow.
			// Drop the event for that consumer and continue.
		}
	}

	return nil
}

func (b *Bus) PublishNow(e Event) error {
	return b.Publish(context.Background(), e)
}

func (s subscription) matches(t Type) bool {
	if s.types == nil {
		return true
	}
	_, ok := s.types[t]
	return ok
}

func (e Event) Validate() error {
	if e.Type == "" {
		return ErrInvalidEvent
	}
	if e.Timestamp.IsZero() {
		return ErrInvalidEvent
	}
	return nil
}

func NewEvent(eventType Type, source string) Event {
	return Event{
		ID:        fmt.Sprintf("%d", time.Now().UTC().UnixNano()),
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().UTC(),
		Data:      map[string]any{},
	}
}