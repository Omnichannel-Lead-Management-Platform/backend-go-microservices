package domain

import "context"

// LeadRepository defines the interface for lead and stage database operations.
type LeadRepository interface {
	// Lead operations
	CreateLead(ctx context.Context, lead *Lead) error
	GetLeadByID(ctx context.Context, workspaceID, leadID string) (*Lead, error)
	ListLeads(ctx context.Context, workspaceID string) ([]*Lead, error)
	UpdateLeadStage(ctx context.Context, workspaceID, leadID, newStage string) error
	AssignLead(ctx context.Context, workspaceID, leadID, userID string) error
	UpdateLeadTags(ctx context.Context, workspaceID, leadID string, tags []string) error
	InsertLeadStateHistory(ctx context.Context, history *LeadStateHistory) error

	// Stage operations
	GetWorkspaceStages(ctx context.Context, workspaceID string) ([]*LeadStage, error)
	CreateLeadStage(ctx context.Context, stage *LeadStage) error
}

// NoteRepository defines the interface for internal note operations.
type NoteRepository interface {
	CreateNote(ctx context.Context, note *InternalNote) error
	GetNotesByConversationID(ctx context.Context, workspaceID, conversationID string) ([]*InternalNote, error)
}

// ReminderRepository defines the interface for follow-up reminder operations.
type ReminderRepository interface {
	CreateReminder(ctx context.Context, reminder *FollowUpReminder) error
	GetPendingReminders(ctx context.Context) ([]*FollowUpReminder, error)
	MarkReminderCompleted(ctx context.Context, reminderID string) error
}
