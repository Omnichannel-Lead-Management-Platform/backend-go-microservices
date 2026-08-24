package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/omnichannel/lead_management_service/internal/domain"
)

// AISummaryWorker is Pillar 5: A background worker that listens for events
// and simulates doing heavy AI processing without blocking the API.
type AISummaryWorker struct {
	subscriber domain.EventSubscriber
}

func NewAISummaryWorker(subscriber domain.EventSubscriber) *AISummaryWorker {
	return &AISummaryWorker{
		subscriber: subscriber,
	}
}

// Start runs the worker in an infinite loop listening for events.
func (w *AISummaryWorker) Start(ctx context.Context) error {
	// Subscribe to the "lead_events" topic (simulating Redis channel)
	ch, err := w.subscriber.Subscribe(ctx, "lead_events")
	if err != nil {
		return fmt.Errorf("failed to subscribe to lead events: %w", err)
	}

	log.Println("🚀 Pillar 5: AI Summary Worker is running in the background and listening for events...")

	// Listen for incoming messages on the channel forever (until app closes)
	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 AI Summary Worker shutting down gracefully...")
			return nil
		case event := <-ch:
			// We received a message! Let's handle it.
			w.handleEvent(event)
		}
	}
}

func (w *AISummaryWorker) handleEvent(event domain.Event) {
	// We only care about stage changes for the AI Summary
	if event.Type != "lead.stage_changed" {
		return
	}

	// This payload was injected in lead_service.go UpdateLeadStage
	payload, ok := event.Payload.(map[string]interface{})
	if !ok {
		log.Printf("Worker Error: Invalid event payload format")
		return
	}

	leadID := payload["lead_id"].(string)
	newStage := payload["new_stage"].(string)

	log.Printf("\n=======================================================")
	log.Printf("🤖 [AI WORKER WOKE UP] Event Received: Lead %s moved to stage '%s'", leadID, newStage)
	
	// Simulate 3 seconds of "Heavy AI Work" reading WhatsApp messages
	log.Printf("⏳ [AI WORKER] Analyzing 42 WhatsApp messages for Lead %s...", leadID)
	time.Sleep(3 * time.Second)
	
	log.Printf("✅ [AI WORKER] AI Summary generated and saved successfully!")
	log.Printf("=======================================================\n")
}
