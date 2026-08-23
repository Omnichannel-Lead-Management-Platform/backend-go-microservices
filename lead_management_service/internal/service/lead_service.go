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
