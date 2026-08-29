package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/omnichannel/lead_management_service/internal/domain"
)

// ActivityWorker listens for inbound message events and bumps the 
// last_activity_at timestamp on the associated lead.
type ActivityWorker struct {
	subscriber domain.EventSubscriber
	repo       domain.LeadRepository
}

func NewActivityWorker(subscriber domain.EventSubscriber, repo domain.LeadRepository) *ActivityWorker {
	return &ActivityWorker{
		subscriber: subscriber,
		repo:       repo,
	}
}

func (w *ActivityWorker) Start(ctx context.Context) error {
	ch, err := w.subscriber.Subscribe(ctx, "message_events")
	if err != nil {
		return fmt.Errorf("failed to subscribe to message events: %w", err)
	}

	log.Println("🚀 Activity Worker is listening for incoming messages...")

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Activity Worker shutting down...")
			return nil
		case event := <-ch:
			w.handleEvent(ctx, event)
		}
	}
}

func (w *ActivityWorker) handleEvent(ctx context.Context, event domain.Event) {
	if event.Type != "message.received" {
		return
	}

	payload, ok := event.Payload.(map[string]interface{})
	if !ok {
		log.Printf("Worker Error: Invalid event payload format")
		return
	}

	workspaceID, ok1 := payload["workspace_id"].(string)
	conversationID, ok2 := payload["conversation_id"].(string)

	if !ok1 || !ok2 {
		log.Printf("Worker Error: Missing workspace_id or conversation_id in payload")
		return
	}

	err := w.repo.UpdateLeadActivity(ctx, workspaceID, conversationID)
	if err != nil {
		log.Printf("Worker Error: Failed to update lead activity: %v", err)
		return
	}

	log.Printf("✅ [ACTIVITY WORKER] Bumbped activity for conversation %s", conversationID)
}
