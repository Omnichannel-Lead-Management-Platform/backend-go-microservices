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
	CreateLeadStageFunc    func(ctx context.Context, stage *domain.LeadStage) error
	GetWorkspaceStagesFunc func(ctx context.Context, workspaceID string) ([]*domain.LeadStage, error)
	AssignLeadFunc         func(ctx context.Context, workspaceID, leadID, userID string) error
	UpdateLeadTagsFunc     func(ctx context.Context, workspaceID, leadID string, tags []string) error
}

func (m *MockLeadRepository) CreateLeadStage(ctx context.Context, stage *domain.LeadStage) error {
	if m.CreateLeadStageFunc != nil {
		return m.CreateLeadStageFunc(ctx, stage)
	}
	return nil
}

func (m *MockLeadRepository) GetWorkspaceStages(ctx context.Context, workspaceID string) ([]*domain.LeadStage, error) {
	if m.GetWorkspaceStagesFunc != nil {
		return m.GetWorkspaceStagesFunc(ctx, workspaceID)
	}
	return nil, nil
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
