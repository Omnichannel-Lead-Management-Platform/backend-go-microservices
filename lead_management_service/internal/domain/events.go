package domain

import "context"

// Event represents a message to be published to a message broker (e.g. Redis Streams).
type Event struct {
	Type        string `json:"type"`         // e.g., "lead.stage_changed"
	WorkspaceID string `json:"workspace_id"`
	Payload     any    `json:"payload"`      // Event-specific data
}

// EventPublisher defines the interface for publishing asynchronous events.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, event Event) error
}

// EventSubscriber defines the interface for listening to asynchronous events.
type EventSubscriber interface {
	Subscribe(ctx context.Context, topic string) (<-chan Event, error)
}
