CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workspace_id, name)
);

CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Seed basic permissions
INSERT INTO permissions (name, description) VALUES
('workspace:manage', 'Can manage workspace settings and billing'),
('users:manage', 'Can invite, remove, and manage roles of other users'),
('roles:manage', 'Can create and modify custom roles'),
('leads:read', 'Can view leads'),
('leads:write', 'Can create and edit leads'),
('leads:delete', 'Can delete leads'),
('messages:read', 'Can read messages'),
('messages:write', 'Can send messages');

-- Add role_id to users and drop the old role string column
ALTER TABLE users ADD COLUMN role_id UUID REFERENCES roles(id) ON DELETE SET NULL;

-- Note: In a real production system with existing data, we would write a migration script 
-- to map the existing 'role' strings to new role_ids. Since this is early development,
-- we'll just drop the column and rely on the new role_id.
ALTER TABLE users DROP COLUMN role;
