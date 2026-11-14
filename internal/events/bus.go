package events

import (
	"context"
	"sync"
)

// InMemoryBus is a simple synchronous in-process event bus.
type InMemoryBus struct {
	mu       sync.RWMutex
	handlers map[Type][]Handler
}

// NewBus creates an in-memory event bus.
func NewBus() *InMemoryBus {
	return &InMemoryBus{handlers: map[Type][]Handler{}}
}

func (b *InMemoryBus) Publish(ctx context.Context, event Event) error {
	b.mu.RLock()
	hs := append([]Handler(nil), b.handlers[event.Type]...)
	b.mu.RUnlock()
	for _, h := range hs {
		if h != nil {
			h(ctx, event)
		}
	}
	return nil
}

func (b *InMemoryBus) Subscribe(eventType Type, handler Handler) func() {
	b.mu.Lock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	idx := len(b.handlers[eventType]) - 1
	b.mu.Unlock()
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		hs := b.handlers[eventType]
		if idx >= 0 && idx < len(hs) {
			hs[idx] = nil
		}
	}
}
