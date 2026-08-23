package domain

import (
	"time"
)

// Lead represents a customer inquiry from a connected channel.
type Lead struct {
	ID             string    `json:"id" db:"id"`
	WorkspaceID    string    `json:"workspace_id" db:"workspace_id"`
	LeadState      string    `json:"lead_state" db:"lead_state"` // Default: 'New'
	FirstName      string    `json:"first_name" db:"first_name"`
	LastName       string    `json:"last_name" db:"last_name"`
	Email          string    `json:"email" db:"email"`
	Phone          string    `json:"phone" db:"phone"`
	AssignedTo     *string   `json:"assigned_to,omitempty" db:"assigned_to"`
	Tags           []string  `json:"tags" db:"tags"` // JSONB array of tags
	LastActivityAt time.Time `json:"last_activity_at" db:"last_activity_at"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// LeadStage represents a configurable pipeline stage for a workspace.
type LeadStage struct {
	ID          string    `json:"id" db:"id"`
	WorkspaceID string    `json:"workspace_id" db:"workspace_id"`
	StageKey    string    `json:"stage_key" db:"stage_key"`
	Label       string    `json:"label" db:"label"`
	Color       string    `json:"color" db:"color"`
	Position    int       `json:"position" db:"position"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// InternalNote represents a note added by an agent, not visible to the customer.
type InternalNote struct {
	ID             string    `json:"id" db:"id"`
	WorkspaceID    string    `json:"workspace_id" db:"workspace_id"`
	ConversationID string    `json:"conversation_id" db:"conversation_id"`
	UserID         string    `json:"user_id" db:"user_id"`
	Content        string    `json:"content" db:"content"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// FollowUpReminder represents a scheduled reminder for a lead.
type FollowUpReminder struct {
	ID          string    `json:"id" db:"id"`
	WorkspaceID string    `json:"workspace_id" db:"workspace_id"`
	LeadID      string    `json:"lead_id" db:"lead_id"`
	AssignedTo  *string   `json:"assigned_to,omitempty" db:"assigned_to"`
	RemindAt    time.Time `json:"remind_at" db:"remind_at"`
	Notes       string    `json:"notes" db:"notes"`
	IsCompleted bool      `json:"is_completed" db:"is_completed"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// LeadStateHistory represents an audit trail record of a lead changing stages.
type LeadStateHistory struct {
	ID          string    `json:"id" db:"id"`
	WorkspaceID string    `json:"workspace_id" db:"workspace_id"`
	LeadID      string    `json:"lead_id" db:"lead_id"`
	FromState   string    `json:"from_state" db:"from_state"`
	ToState     string    `json:"to_state" db:"to_state"`
	ChangedBy   *string   `json:"changed_by,omitempty" db:"changed_by"`
	ChangedAt   time.Time `json:"changed_at" db:"changed_at"`
}

// MessageTemplate represents a saved reply for quick responses.
type MessageTemplate struct {
	ID          string    `json:"id" db:"id"`
	WorkspaceID string    `json:"workspace_id" db:"workspace_id"`
	Title       string    `json:"title" db:"title"`
	Content     string    `json:"content" db:"content"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
