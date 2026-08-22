package postgres

import (
	"context"
	"database/sql"
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
		INSERT INTO leads (workspace_id, lead_state, first_name, last_name, email, phone)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		lead.WorkspaceID,
		lead.LeadState,
		lead.FirstName,
		lead.LastName,
		lead.Email,
		lead.Phone,
	).Scan(&lead.ID, &lead.CreatedAt, &lead.UpdatedAt)

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
		SELECT id, workspace_id, lead_state, first_name, last_name, email, phone, created_at, updated_at
		FROM leads
		WHERE workspace_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query leads: %w", err)
	}
	defer rows.Close()

	var leads []*domain.Lead
	for rows.Next() {
		var l domain.Lead
		err := rows.Scan(
			&l.ID, &l.WorkspaceID, &l.LeadState, &l.FirstName, &l.LastName,
			&l.Email, &l.Phone, &l.CreatedAt, &l.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lead row: %w", err)
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
	// Implementation omitted for brevity
	return nil
}

// GetWorkspaceStages fetches custom stages for a workspace.
func (r *LeadRepository) GetWorkspaceStages(ctx context.Context, workspaceID string) ([]*domain.LeadStage, error) {
	// Implementation omitted for brevity
	return nil, nil
}

// CreateLeadStage inserts a new custom stage.
func (r *LeadRepository) CreateLeadStage(ctx context.Context, stage *domain.LeadStage) error {
	// Implementation omitted for brevity
	return nil
}
