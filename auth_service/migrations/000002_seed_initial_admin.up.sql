CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Insert default workspace and get its ID to link the admin user to it
WITH new_workspace AS (
    INSERT INTO workspaces (name)
    VALUES ('Default Workspace')
    RETURNING id
)
INSERT INTO users (workspace_id, role, name, email, password, "emailVerified")
SELECT 
    id, 
    'admin', 
    'Admin User', 
    'admin@admin.com', 
    crypt('password123', gen_salt('bf', 10)), 
    true
FROM new_workspace;
