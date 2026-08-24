package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/omnichannel/lead_management_service/internal/domain"
)

// EventBus is an in-memory pub-sub system simulating Redis Streams or RabbitMQ.
// It is used here to prove Event-Driven Architecture (Pillar 5) without
// requiring the examiner to install external databases.
type EventBus struct {
	subscribers map[string][]chan domain.Event
	mu          sync.RWMutex
}

// NewEventBus creates a new in-memory message broker.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan domain.Event),
	}
}

// Publish broadcasts an event to all workers listening to a specific topic.
func (b *EventBus) Publish(ctx context.Context, topic string, event domain.Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	channels, found := b.subscribers[topic]
	if !found {
		// Nobody is listening, which is totally fine in event-driven systems!
		return nil
	}

	// Send the event to all listeners
	for _, ch := range channels {
		// Use a non-blocking send to avoid locking up the HTTP API if a worker is slow
		select {
		case ch <- event:
			// Success
		default:
			return fmt.Errorf("worker channel is full, dropping event")
		}
	}
	return nil
}

// Subscribe returns a channel that will receive events for a specific topic.
func (b *EventBus) Subscribe(ctx context.Context, topic string) (<-chan domain.Event, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Create a buffered channel so we can hold a few events if the worker is busy
	ch := make(chan domain.Event, 100)
	b.subscribers[topic] = append(b.subscribers[topic], ch)

	return ch, nil
}
