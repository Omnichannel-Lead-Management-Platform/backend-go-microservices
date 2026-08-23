CREATE TABLE IF NOT EXISTS lead_state_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    from_state VARCHAR(100) NOT NULL,
    to_state VARCHAR(100) NOT NULL,
    changed_by UUID, -- Can be null if changed by system/API automatically without a user token
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index to quickly fetch history for a specific lead
CREATE INDEX idx_lead_state_history_lead_id ON lead_state_history(lead_id);
-- Index to ensure fast multi-tenant queries
CREATE INDEX idx_lead_state_history_workspace_id ON lead_state_history(workspace_id);
