package service

import (
	"context"
	"errors"
	"testing"

	"github.com/omnichannel/lead_management_service/internal/domain"
)

// ==========================================
// 1. THE FAKE DATABASE (Mock Repository)
// ==========================================
// This is our "Fake Database Assistant". We use this instead of PostgreSQL
// so our tests run instantly and don't require a real database connection.
type MockLeadRepository struct {
	// We embed the interface so we don't have to write dummy functions 
	// for the ones we aren't testing today.
	domain.LeadRepository

	// These variables let us control what the fake database replies with
	CreateLeadFunc             func(ctx context.Context, lead *domain.Lead) error
	UpdateLeadStageFunc        func(ctx context.Context, workspaceID, leadID, newStage string) error
	AssignLeadFunc             func(ctx context.Context, workspaceID, leadID, userID string) error
	UpdateLeadTagsFunc         func(ctx context.Context, workspaceID, leadID string, tags []string) error
	InsertLeadStateHistoryFunc func(ctx context.Context, history *domain.LeadStateHistory) error
	UpdateLeadActivityFunc     func(ctx context.Context, workspaceID, conversationID string) error

	// Stage operations
	GetWorkspaceStagesFunc       func(ctx context.Context, workspaceID string) ([]*domain.LeadStage, error)
	CreateLeadStageFunc          func(ctx context.Context, stage *domain.LeadStage) error
	UpdateLeadStageConfigFunc    func(ctx context.Context, stage *domain.LeadStage) error
	DeleteLeadStageFunc          func(ctx context.Context, workspaceID, stageID string) error
	UpdateLeadStagePositionsFunc func(ctx context.Context, workspaceID string, stageIDs []string) error
	
	// Pillar 3
	AddInternalNoteFunc          func(ctx context.Context, note *domain.InternalNote) error
	GetInternalNotesByLeadFunc   func(ctx context.Context, workspaceID, leadID string) ([]*domain.InternalNote, error)

	// Pillar 4
	CreateMessageTemplateFunc    func(ctx context.Context, template *domain.MessageTemplate) error
	GetMessageTemplatesFunc      func(ctx context.Context, workspaceID string) ([]*domain.MessageTemplate, error)
	UpdateMessageTemplateFunc    func(ctx context.Context, template *domain.MessageTemplate) error
	DeleteMessageTemplateFunc    func(ctx context.Context, workspaceID, templateID string) error
}

func (m *MockLeadRepository) CreateLead(ctx context.Context, lead *domain.Lead) error {
	if m.CreateLeadFunc != nil {
		return m.CreateLeadFunc(ctx, lead)
	}
	return nil
}

func (m *MockLeadRepository) UpdateLeadStage(ctx context.Context, workspaceID, leadID, newStage string) error {
	if m.UpdateLeadStageFunc != nil {
		return m.UpdateLeadStageFunc(ctx, workspaceID, leadID, newStage)
	}
	return nil
}

func (m *MockLeadRepository) GetWorkspaceStages(ctx context.Context, workspaceID string) ([]*domain.LeadStage, error) {
	if m.GetWorkspaceStagesFunc != nil {
		return m.GetWorkspaceStagesFunc(ctx, workspaceID)
	}
	return nil, nil
}

func (m *MockLeadRepository) CreateLeadStage(ctx context.Context, stage *domain.LeadStage) error {
	if m.CreateLeadStageFunc != nil {
		return m.CreateLeadStageFunc(ctx, stage)
	}
	return nil
}

func (m *MockLeadRepository) UpdateLeadStageConfig(ctx context.Context, stage *domain.LeadStage) error {
	if m.UpdateLeadStageConfigFunc != nil {
		return m.UpdateLeadStageConfigFunc(ctx, stage)
	}
	return nil
}

func (m *MockLeadRepository) DeleteLeadStage(ctx context.Context, workspaceID, stageID string) error {
	if m.DeleteLeadStageFunc != nil {
		return m.DeleteLeadStageFunc(ctx, workspaceID, stageID)
	}
	return nil
}

func (m *MockLeadRepository) UpdateLeadStagePositions(ctx context.Context, workspaceID string, stageIDs []string) error {
	if m.UpdateLeadStagePositionsFunc != nil {
		return m.UpdateLeadStagePositionsFunc(ctx, workspaceID, stageIDs)
	}
	return nil
}

func (m *MockLeadRepository) AssignLead(ctx context.Context, workspaceID, leadID, userID string) error {
	if m.AssignLeadFunc != nil {
		return m.AssignLeadFunc(ctx, workspaceID, leadID, userID)
	}
	return nil
}

func (m *MockLeadRepository) UpdateLeadTags(ctx context.Context, workspaceID, leadID string, tags []string) error {
	if m.UpdateLeadTagsFunc != nil {
		return m.UpdateLeadTagsFunc(ctx, workspaceID, leadID, tags)
	}
	return nil
}

func (m *MockLeadRepository) UpdateLeadActivity(ctx context.Context, workspaceID, conversationID string) error {
	if m.UpdateLeadActivityFunc != nil {
		return m.UpdateLeadActivityFunc(ctx, workspaceID, conversationID)
	}
	return nil
}

func (m *MockLeadRepository) InsertLeadStateHistory(ctx context.Context, history *domain.LeadStateHistory) error {
	if m.InsertLeadStateHistoryFunc != nil {
		return m.InsertLeadStateHistoryFunc(ctx, history)
	}
	return nil
}

func (m *MockLeadRepository) AddInternalNote(ctx context.Context, note *domain.InternalNote) error {
	if m.AddInternalNoteFunc != nil {
		return m.AddInternalNoteFunc(ctx, note)
	}
	return nil
}

func (m *MockLeadRepository) GetInternalNotesByLead(ctx context.Context, workspaceID, leadID string) ([]*domain.InternalNote, error) {
	if m.GetInternalNotesByLeadFunc != nil {
		return m.GetInternalNotesByLeadFunc(ctx, workspaceID, leadID)
	}
	return nil, nil
}

// Pillar 4 Mock Methods

func (m *MockLeadRepository) CreateMessageTemplate(ctx context.Context, template *domain.MessageTemplate) error {
	if m.CreateMessageTemplateFunc != nil {
		return m.CreateMessageTemplateFunc(ctx, template)
	}
	return nil
}

func (m *MockLeadRepository) GetMessageTemplates(ctx context.Context, workspaceID string) ([]*domain.MessageTemplate, error) {
	if m.GetMessageTemplatesFunc != nil {
		return m.GetMessageTemplatesFunc(ctx, workspaceID)
	}
	return nil, nil
}

func (m *MockLeadRepository) UpdateMessageTemplate(ctx context.Context, template *domain.MessageTemplate) error {
	if m.UpdateMessageTemplateFunc != nil {
		return m.UpdateMessageTemplateFunc(ctx, template)
	}
	return nil
}

func (m *MockLeadRepository) DeleteMessageTemplate(ctx context.Context, workspaceID, templateID string) error {
	if m.DeleteMessageTemplateFunc != nil {
		return m.DeleteMessageTemplateFunc(ctx, workspaceID, templateID)
	}
	return nil
}

// A fake event publisher so NewLeadService doesn't crash
type MockEventPublisher struct{}
func (m *MockEventPublisher) Publish(ctx context.Context, topic string, event domain.Event) error {
	return nil
}

// ==========================================
// 2. THE ACTUAL TEST CASES
// ==========================================

func TestCreateLeadStage_Success(t *testing.T) {
	// 1. Set up the Fake Database
	fakeRepo := &MockLeadRepository{
		CreateLeadStageFunc: func(ctx context.Context, stage *domain.LeadStage) error {
			// Pretend the database saved it successfully and assigned an ID
			stage.ID = "fake-uuid-123"
			return nil
		},
	}
	fakePublisher := &MockEventPublisher{}

	// 2. Hire the Head Chef (Service) and give them the fake database
	service := NewLeadService(fakeRepo, fakePublisher)

	// 3. Create a fake stage order
	newStage := &domain.LeadStage{
		WorkspaceID: "ws-123",
		Label:       "New Lead",
	}

	// 4. Run the Chef's function!
	err := service.CreateLeadStage(context.Background(), newStage)

	// 5. Check the results (Assertions)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	if newStage.ID != "fake-uuid-123" {
		t.Errorf("Expected stage to be given an ID by the database, got empty")
	}
}

func TestCreateLeadStage_DatabaseError(t *testing.T) {
	// 1. Set up a Fake Database that is broken!
	fakeRepo := &MockLeadRepository{
		CreateLeadStageFunc: func(ctx context.Context, stage *domain.LeadStage) error {
			return errors.New("database connection refused")
		},
	}
	fakePublisher := &MockEventPublisher{}
	service := NewLeadService(fakeRepo, fakePublisher)

	newStage := &domain.LeadStage{WorkspaceID: "ws-123"}

	// 2. Run the Chef's function
	err := service.CreateLeadStage(context.Background(), newStage)

	// 3. We EXPECT an error this time!
	if err == nil {
		t.Errorf("Expected an error because the database is broken, but got none!")
	}
}

// ---- PILLAR 2 TESTS ----

func TestAssignLead_Success(t *testing.T) {
	fakeRepo := &MockLeadRepository{
		AssignLeadFunc: func(ctx context.Context, workspaceID, leadID, userID string) error {
			return nil // DB succeeded!
		},
	}
	service := NewLeadService(fakeRepo, &MockEventPublisher{})

	err := service.AssignLead(context.Background(), "ws-1", "lead-1", "user-bob")
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
}

func TestUpdateLeadTags_TooManyTags(t *testing.T) {
	fakeRepo := &MockLeadRepository{} // Database won't even be called
	service := NewLeadService(fakeRepo, &MockEventPublisher{})

	tags := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}
	
	err := service.UpdateLeadTags(context.Background(), "ws-1", "lead-1", tags)
	if err == nil {
		t.Errorf("Expected an error for more than 10 tags, but got none!")
	}
}

// ---- PILLAR 3 TESTS ----

func TestAddInternalNote_Success(t *testing.T) {
	fakeRepo := &MockLeadRepository{
		AddInternalNoteFunc: func(ctx context.Context, note *domain.InternalNote) error {
			note.ID = "note-123"
			return nil
		},
	}
	service := NewLeadService(fakeRepo, &MockEventPublisher{})

	note := &domain.InternalNote{
		WorkspaceID: "ws-1",
		Content:     "This is a valid note!",
	}

	err := service.AddInternalNote(context.Background(), note)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if note.ID != "note-123" {
		t.Errorf("Expected database to set note ID, got empty")
	}
}

func TestAddInternalNote_EmptyContent(t *testing.T) {
	fakeRepo := &MockLeadRepository{} // DB won't be called because Service blocks it
	service := NewLeadService(fakeRepo, &MockEventPublisher{})

	note := &domain.InternalNote{
		WorkspaceID: "ws-1",
		Content:     "", // Invalid!
	}

	err := service.AddInternalNote(context.Background(), note)
	if err == nil {
		t.Errorf("Expected Head Chef to block empty notes, but it succeeded!")
	}
}

// ---- PILLAR 4 TESTS ----

func TestCreateMessageTemplate_Success(t *testing.T) {
	fakeRepo := &MockLeadRepository{
		CreateMessageTemplateFunc: func(ctx context.Context, template *domain.MessageTemplate) error {
			template.ID = "tpl-123"
			return nil
		},
	}
	service := NewLeadService(fakeRepo, &MockEventPublisher{})

	template := &domain.MessageTemplate{
		WorkspaceID: "ws-1",
		Title:       "Greeting",
		Content:     "Hello there!",
	}

	err := service.CreateMessageTemplate(context.Background(), template)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if template.ID != "tpl-123" {
		t.Errorf("Expected database to set template ID, got empty")
	}
}

func TestCreateMessageTemplate_EmptyTitle(t *testing.T) {
	fakeRepo := &MockLeadRepository{} // Database won't be called
	service := NewLeadService(fakeRepo, &MockEventPublisher{})

	template := &domain.MessageTemplate{
		WorkspaceID: "ws-1",
		Title:       "", // Invalid!
		Content:     "Hello there!",
	}

	err := service.CreateMessageTemplate(context.Background(), template)
	if err == nil {
		t.Errorf("Expected Service to block empty title, but it succeeded!")
	}
}

// ---- NEW TESTS FOR PIPELINE CONFIG ----

func TestUpdateLeadStageConfig_EmptyLabel(t *testing.T) {
	fakeRepo := &MockLeadRepository{}
	service := NewLeadService(fakeRepo, &MockEventPublisher{})

	stage := &domain.LeadStage{
		WorkspaceID: "ws-1",
		ID:          "stg-1",
		Label:       "", // Invalid!
	}

	err := service.UpdateLeadStageConfig(context.Background(), stage)
	if err == nil {
		t.Errorf("Expected Service to block empty label, but it succeeded!")
	}
}

func TestReorderLeadStages_EmptyList(t *testing.T) {
	fakeRepo := &MockLeadRepository{}
	service := NewLeadService(fakeRepo, &MockEventPublisher{})

	err := service.ReorderLeadStages(context.Background(), "ws-1", []string{})
	if err == nil {
		t.Errorf("Expected Service to block empty stage list, but it succeeded!")
	}
}
