CREATE TABLE IF NOT EXISTS lead_stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    stage_key VARCHAR(50) NOT NULL,
    label VARCHAR(100) NOT NULL,
    color VARCHAR(20) DEFAULT '#3B82F6',
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workspace_id, stage_key)
);

-- Indexing for multi-tenancy and ordering performance
CREATE INDEX IF NOT EXISTS idx_lead_stages_workspace ON lead_stages(workspace_id);
CREATE INDEX IF NOT EXISTS idx_lead_stages_workspace_position ON lead_stages(workspace_id, position);
