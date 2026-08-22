ALTER TABLE users ADD COLUMN role VARCHAR(50) NOT NULL DEFAULT 'agent';
ALTER TABLE users DROP COLUMN role_id;

DROP TABLE role_permissions;
DROP TABLE roles;
DROP TABLE permissions;
