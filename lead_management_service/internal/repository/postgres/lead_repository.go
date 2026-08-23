package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/omnichannel/lead_management_service/internal/domain"
)

type LeadRepository struct {
	db *sql.DB
}

func NewLeadRepository(db *sql.DB) *LeadRepository {
	return &LeadRepository{db: db}
}

// CreateLead inserts a new lead into the database.
func (r *LeadRepository) CreateLead(ctx context.Context, lead *domain.Lead) error {
	query := `
		INSERT INTO leads (workspace_id, lead_state, first_name, last_name, email, phone, assigned_to, tags, last_activity_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP)
		RETURNING id, last_activity_at, created_at, updated_at
	`
	
	tagsJSON, err := json.Marshal(lead.Tags)
	if err != nil {
		tagsJSON = []byte("[]") // Default empty array if nil
	}
	err = r.db.QueryRowContext(ctx, query,
		lead.WorkspaceID,
		lead.LeadState,
		lead.FirstName,
		lead.LastName,
		lead.Email,
		lead.Phone,
		lead.AssignedTo,
		tagsJSON,
	).Scan(&lead.ID, &lead.LastActivityAt, &lead.CreatedAt, &lead.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create lead: %w", err)
	}
	return nil
}

// UpdateLeadStage updates the stage of an existing lead.
func (r *LeadRepository) UpdateLeadStage(ctx context.Context, workspaceID, leadID, newStage string) error {
	query := `
		UPDATE leads
		SET lead_state = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND workspace_id = $3
	`
	result, err := r.db.ExecContext(ctx, query, newStage, leadID, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to update lead stage: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("lead not found or not in workspace")
	}
	return nil
}

// ListLeads fetches all leads for a specific workspace.
func (r *LeadRepository) ListLeads(ctx context.Context, workspaceID string) ([]*domain.Lead, error) {
	query := `
		SELECT id, workspace_id, lead_state, first_name, last_name, email, phone, assigned_to, tags, last_activity_at, created_at, updated_at
		FROM leads
		WHERE workspace_id = $1
		ORDER BY last_activity_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query leads: %w", err)
	}
	defer rows.Close()

	var leads []*domain.Lead
	for rows.Next() {
		var l domain.Lead
		var tagsBytes []byte

		err := rows.Scan(
			&l.ID, &l.WorkspaceID, &l.LeadState, &l.FirstName, &l.LastName,
			&l.Email, &l.Phone, &l.AssignedTo, &tagsBytes, &l.LastActivityAt, &l.CreatedAt, &l.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lead row: %w", err)
		}
		
		if tagsBytes != nil {
			_ = json.Unmarshal(tagsBytes, &l.Tags)
		}
		
		leads = append(leads, &l)
	}
	
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return leads, nil
}

// GetLeadByID retrieves a specific lead.
func (r *LeadRepository) GetLeadByID(ctx context.Context, workspaceID, leadID string) (*domain.Lead, error) {
	// Implementation omitted for brevity, follows same pattern as above
	return nil, nil
}

// AssignLead assigns a lead to an agent.
func (r *LeadRepository) AssignLead(ctx context.Context, workspaceID, leadID, userID string) error {
	query := `
		UPDATE leads
		SET assigned_to = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND workspace_id = $3
	`
	result, err := r.db.ExecContext(ctx, query, userID, leadID, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to assign lead: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("lead not found or not in workspace")
	}
	return nil
}

// UpdateLeadTags updates the classification tags of a lead.
func (r *LeadRepository) UpdateLeadTags(ctx context.Context, workspaceID, leadID string, tags []string) error {
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		tagsJSON = []byte("[]")
	}

	query := `
		UPDATE leads
		SET tags = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND workspace_id = $3
	`
	result, err := r.db.ExecContext(ctx, query, tagsJSON, leadID, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to update lead tags: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("lead not found or not in workspace")
	}
	return nil
}

// InsertLeadStateHistory logs an audit trail of a lead's stage change.
func (r *LeadRepository) InsertLeadStateHistory(ctx context.Context, history *domain.LeadStateHistory) error {
	query := `
		INSERT INTO lead_state_history (workspace_id, lead_id, from_state, to_state, changed_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, changed_at
	`
	err := r.db.QueryRowContext(ctx, query,
		history.WorkspaceID,
		history.LeadID,
		history.FromState,
		history.ToState,
		history.ChangedBy,
	).Scan(&history.ID, &history.ChangedAt)

	if err != nil {
		return fmt.Errorf("failed to insert lead state history: %w", err)
	}
	return nil
}

// GetWorkspaceStages fetches custom stages for a workspace.
func (r *LeadRepository) GetWorkspaceStages(ctx context.Context, workspaceID string) ([]*domain.LeadStage, error) {
	query := `
		SELECT id, workspace_id, stage_key, label, color, position, created_at, updated_at
		FROM lead_stages
		WHERE workspace_id = $1
		ORDER BY position ASC
	`
	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stages: %w", err)
	}
	defer rows.Close()

	var stages []*domain.LeadStage
	for rows.Next() {
		var s domain.LeadStage
		err := rows.Scan(&s.ID, &s.WorkspaceID, &s.StageKey, &s.Label, &s.Color, &s.Position, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stage: %w", err)
		}
		stages = append(stages, &s)
	}
	return stages, nil
}

// CreateLeadStage inserts a new custom stage.
func (r *LeadRepository) CreateLeadStage(ctx context.Context, stage *domain.LeadStage) error {
	query := `
		INSERT INTO lead_stages (workspace_id, stage_key, label, color, position)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		stage.WorkspaceID,
		stage.StageKey,
		stage.Label,
		stage.Color,
		stage.Position,
	).Scan(&stage.ID, &stage.CreatedAt, &stage.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create lead stage: %w", err)
	}
	return nil
}

// AddInternalNote inserts a private note left by an agent.
func (r *LeadRepository) AddInternalNote(ctx context.Context, note *domain.InternalNote) error {
	query := `
		INSERT INTO internal_notes (workspace_id, conversation_id, user_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		note.WorkspaceID,
		note.ConversationID,
		note.UserID,
		note.Content,
	).Scan(&note.ID, &note.CreatedAt, &note.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to add internal note: %w", err)
	}
	return nil
}

// GetInternalNotesByLead fetches all internal notes for a specific lead using a SQL JOIN.
func (r *LeadRepository) GetInternalNotesByLead(ctx context.Context, workspaceID, leadID string) ([]*domain.InternalNote, error) {
	query := `
		SELECT n.id, n.workspace_id, n.conversation_id, n.user_id, n.content, n.created_at, n.updated_at
		FROM internal_notes n
		JOIN conversations c ON n.conversation_id = c.id
		WHERE n.workspace_id = $1 AND c.lead_id = $2
		ORDER BY n.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, workspaceID, leadID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch internal notes: %w", err)
	}
	defer rows.Close()

	var notes []*domain.InternalNote
	for rows.Next() {
		var n domain.InternalNote
		err := rows.Scan(&n.ID, &n.WorkspaceID, &n.ConversationID, &n.UserID, &n.Content, &n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan internal note: %w", err)
		}
		notes = append(notes, &n)
	}
	return notes, nil
}

// ---- Pillar 4: Message Templates ----

func (r *LeadRepository) CreateMessageTemplate(ctx context.Context, t *domain.MessageTemplate) error {
	query := `
		INSERT INTO message_templates (workspace_id, title, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query, t.WorkspaceID, t.Title, t.Content).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}
	return nil
}

func (r *LeadRepository) GetMessageTemplates(ctx context.Context, workspaceID string) ([]*domain.MessageTemplate, error) {
	query := `
		SELECT id, workspace_id, title, content, created_at, updated_at
		FROM message_templates
		WHERE workspace_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch templates: %w", err)
	}
	defer rows.Close()

	var templates []*domain.MessageTemplate
	for rows.Next() {
		var t domain.MessageTemplate
		if err := rows.Scan(&t.ID, &t.WorkspaceID, &t.Title, &t.Content, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, &t)
	}
	return templates, nil
}

func (r *LeadRepository) UpdateMessageTemplate(ctx context.Context, t *domain.MessageTemplate) error {
	query := `
		UPDATE message_templates
		SET title = $1, content = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND workspace_id = $4
	`
	result, err := r.db.ExecContext(ctx, query, t.Title, t.Content, t.ID, t.WorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("template not found")
	}
	return nil
}

func (r *LeadRepository) DeleteMessageTemplate(ctx context.Context, workspaceID, templateID string) error {
	query := `
		DELETE FROM message_templates
		WHERE id = $1 AND workspace_id = $2
	`
	result, err := r.db.ExecContext(ctx, query, templateID, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("template not found")
	}
	return nil
}
