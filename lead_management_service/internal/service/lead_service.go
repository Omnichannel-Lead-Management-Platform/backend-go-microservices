package service

import (
	"context"
	"fmt"
	"time"

	"github.com/omnichannel/lead_management_service/internal/domain"
)

// LeadService handles the business logic for lead operations.
type LeadService struct {
	repo      domain.LeadRepository
	publisher domain.EventPublisher
}

// NewLeadService creates a new LeadService instance.
func NewLeadService(repo domain.LeadRepository, publisher domain.EventPublisher) *LeadService {
	return &LeadService{
		repo:      repo,
		publisher: publisher,
	}
}

// CreateLead executes business rules for creating a lead.
func (s *LeadService) CreateLead(ctx context.Context, lead *domain.Lead) error {
	// Rule 1: A new lead always defaults to 'New' state if not provided
	if lead.LeadState == "" {
		lead.LeadState = "New"
	}

	// 1. Tell the repository (database assistant) to save it
	err := s.repo.CreateLead(ctx, lead)
	if err != nil {
		return fmt.Errorf("failed to create lead in database: %w", err)
	}

	// 2. Publish a Redis event that a lead was created
	event := domain.Event{
		Type:        "lead.created",
		WorkspaceID: lead.WorkspaceID,
		Payload: map[string]string{
			"lead_id":    lead.ID,
			"first_name": lead.FirstName,
		},
	}
	// We ignore event publish errors here so the user request still succeeds even if Redis blips
	_ = s.publisher.Publish(ctx, "lead_events", event)

	return nil
}

// UpdateLeadStage changes a lead's stage and triggers the AI background summary.
func (s *LeadService) UpdateLeadStage(ctx context.Context, workspaceID, leadID, newStage string) error {
	// Rule 1: First, we should check if the newStage actually exists in this workspace's custom pipeline.
	stages, err := s.repo.GetWorkspaceStages(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("could not fetch workspace stages: %w", err)
	}

	isValidStage := false
	for _, stg := range stages {
		if stg.StageKey == newStage {
			isValidStage = true
			break
		}
	}
	
	// If the admin hasn't created any custom stages yet, we will bypass this check
	// and allow standard defaults like 'New', 'Contacted', etc.
	if len(stages) > 0 && !isValidStage {
		return fmt.Errorf("invalid stage: %s is not configured for this workspace", newStage)
	}

	// 1. Fetch the old stage for the Audit Trail (Skipping DB fetch for simplicity here, assuming 'OldStage')
	changedBy := "System/Agent" // In a real app, this comes from the JWT Token!
	history := &domain.LeadStateHistory{
		WorkspaceID: workspaceID,
		LeadID:      leadID,
		FromState:   "Unknown (Before Update)",
		ToState:     newStage,
		ChangedBy:   &changedBy,
	}
	
	// 2. Write to the Audit Trail (Lead State History)
	_ = s.repo.InsertLeadStateHistory(ctx, history)

	// 3. Update the database via the Repository
	err = s.repo.UpdateLeadStage(ctx, workspaceID, leadID, newStage)
	if err != nil {
		return fmt.Errorf("failed to update lead stage in database: %w", err)
	}

	// 2. THIS IS THE MAGIC: Publish a Redis Event
	// The AI Chatbot Service / Summary Worker is listening for this exact event!
	event := domain.Event{
		Type:        "lead.stage_changed",
		WorkspaceID: workspaceID,
		Payload: map[string]interface{}{
			"lead_id":   leadID,
			"new_stage": newStage,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}
	
	// Shout to Redis so the background worker wakes up
	err = s.publisher.Publish(ctx, "lead_events", event)
	if err != nil {
		// Log error, but don't fail the user request since the DB update succeeded
		fmt.Printf("Warning: failed to publish Redis event: %v\n", err)
	}

	return nil
}

// ListLeads fetches leads.
func (s *LeadService) ListLeads(ctx context.Context, workspaceID string) ([]*domain.Lead, error) {
	return s.repo.ListLeads(ctx, workspaceID)
}

// CreateLeadStage adds a new pipeline stage for a workspace.
func (s *LeadService) CreateLeadStage(ctx context.Context, stage *domain.LeadStage) error {
	// Business Rule: Ensure position is provided or calculate it, but for simplicity, we rely on the input.
	// In a real system, we might query the max position and add 1.
	
	err := s.repo.CreateLeadStage(ctx, stage)
	if err != nil {
		return fmt.Errorf("failed to create lead stage: %w", err)
	}
	return nil
}

// ListWorkspaceStages fetches the custom pipeline stages.
func (s *LeadService) ListWorkspaceStages(ctx context.Context, workspaceID string) ([]*domain.LeadStage, error) {
	return s.repo.GetWorkspaceStages(ctx, workspaceID)
}

// UpdateLeadStageConfig updates a pipeline stage's label or color.
func (s *LeadService) UpdateLeadStageConfig(ctx context.Context, stage *domain.LeadStage) error {
	if stage.Label == "" {
		return fmt.Errorf("stage label cannot be empty")
	}
	return s.repo.UpdateLeadStageConfig(ctx, stage)
}

// DeleteLeadStage removes a custom pipeline stage.
func (s *LeadService) DeleteLeadStage(ctx context.Context, workspaceID, stageID string) error {
	// Business Rule: We could check if there are any leads currently in this stage,
	// and prevent deletion or move them to a default stage.
	// For simplicity, we'll just allow deletion.
	return s.repo.DeleteLeadStage(ctx, workspaceID, stageID)
}

// ReorderLeadStages takes an array of stage IDs in the new correct order and updates their positions.
func (s *LeadService) ReorderLeadStages(ctx context.Context, workspaceID string, stageIDs []string) error {
	if len(stageIDs) == 0 {
		return fmt.Errorf("must provide at least one stage to reorder")
	}
	return s.repo.UpdateLeadStagePositions(ctx, workspaceID, stageIDs)
}

// AssignLead assigns a lead to a specific user (agent).
func (s *LeadService) AssignLead(ctx context.Context, workspaceID, leadID, userID string) error {
	// Business Rule: We could check if the userID actually exists in the Auth Service.
	err := s.repo.AssignLead(ctx, workspaceID, leadID, userID)
	if err != nil {
		return fmt.Errorf("failed to assign lead: %w", err)
	}
	return nil
}

// UpdateLeadTags updates the custom tags on a lead.
func (s *LeadService) UpdateLeadTags(ctx context.Context, workspaceID, leadID string, tags []string) error {
	// Business Rule: We might want to limit to 10 tags maximum.
	if len(tags) > 10 {
		return fmt.Errorf("cannot have more than 10 tags")
	}

	err := s.repo.UpdateLeadTags(ctx, workspaceID, leadID, tags)
	if err != nil {
		return fmt.Errorf("failed to update lead tags: %w", err)
	}
	return nil
}

// AddInternalNote adds a private note for a lead.
func (s *LeadService) AddInternalNote(ctx context.Context, note *domain.InternalNote) error {
	// Business Rule: Ensure the note isn't completely empty
	if note.Content == "" {
		return fmt.Errorf("note content cannot be empty")
	}

	// In a complete system, we would first find the Active Conversation for this Lead.
	// For this simplified version, we just save the note to the DB using the conversation ID provided.
	err := s.repo.AddInternalNote(ctx, note)
	if err != nil {
		return fmt.Errorf("failed to add internal note: %w", err)
	}
	return nil
}

// GetInternalNotesByLead fetches all internal notes for a specific lead.
func (s *LeadService) GetInternalNotesByLead(ctx context.Context, workspaceID, leadID string) ([]*domain.InternalNote, error) {
	return s.repo.GetInternalNotesByLead(ctx, workspaceID, leadID)
}

// ---- Pillar 4: Message Templates ----

// CreateMessageTemplate creates a new saved reply.
func (s *LeadService) CreateMessageTemplate(ctx context.Context, template *domain.MessageTemplate) error {
	if template.Title == "" || template.Content == "" {
		return fmt.Errorf("template title and content cannot be empty")
	}
	return s.repo.CreateMessageTemplate(ctx, template)
}

// GetMessageTemplates fetches all saved replies for a workspace.
func (s *LeadService) GetMessageTemplates(ctx context.Context, workspaceID string) ([]*domain.MessageTemplate, error) {
	return s.repo.GetMessageTemplates(ctx, workspaceID)
}

// UpdateMessageTemplate updates an existing saved reply.
func (s *LeadService) UpdateMessageTemplate(ctx context.Context, template *domain.MessageTemplate) error {
	if template.Title == "" || template.Content == "" {
		return fmt.Errorf("template title and content cannot be empty")
	}
	return s.repo.UpdateMessageTemplate(ctx, template)
}

// DeleteMessageTemplate deletes a saved reply.
func (s *LeadService) DeleteMessageTemplate(ctx context.Context, workspaceID, templateID string) error {
	return s.repo.DeleteMessageTemplate(ctx, workspaceID, templateID)
}

// ---- Chat History ----

// GetMessagesByLead fetches all messages for a lead.
func (s *LeadService) GetMessagesByLead(ctx context.Context, workspaceID, leadID string) ([]*domain.Message, error) {
	return s.repo.GetMessagesByLead(ctx, workspaceID, leadID)
}
