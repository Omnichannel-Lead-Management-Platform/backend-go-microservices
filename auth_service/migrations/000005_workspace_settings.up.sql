ALTER TABLE workspaces
ADD COLUMN settings JSONB DEFAULT '{}'::jsonb;
