-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetWorkspaceByID :one
SELECT * FROM workspaces
WHERE id = $1 LIMIT 1;

-- name: CreateWorkspace :one
INSERT INTO workspaces (name)
VALUES ($1)
RETURNING *;

-- name: CreateUser :one
INSERT INTO users (workspace_id, role_id, name, email, password, "emailVerified")
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateUserTokens :one
UPDATE users 
SET recover_token = $2, recover_token_expiry = $3, confirm_token = $4
WHERE id = $1
RETURNING *;

-- name: UpdateUserPassword :one
UPDATE users
SET password = $2
WHERE id = $1
RETURNING *;

-- name: CreateRole :one
INSERT INTO roles (workspace_id, name, description, is_system)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetRoleByName :one
SELECT * FROM roles
WHERE workspace_id = $1 AND name = $2 LIMIT 1;

-- name: GetUserPermissions :many
SELECT p.name 
FROM users u
JOIN roles r ON u.role_id = r.id
JOIN role_permissions rp ON r.id = rp.role_id
JOIN permissions p ON rp.permission_id = p.id
WHERE u.id = $1;

-- name: AssignPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GetPermissionByName :one
SELECT * FROM permissions
WHERE name = $1 LIMIT 1;


-- name: GetUsersByWorkspaceID :many
SELECT id, workspace_id, role_id, name, email, "emailVerified", image, "createdAt", "updatedAt"
FROM users
WHERE workspace_id = $1;

-- name: UpdateUserRole :exec
UPDATE users
SET role_id = $2
WHERE id = $1 AND workspace_id = $3;

-- name: GetRolesByWorkspaceID :many
SELECT * FROM roles
WHERE workspace_id = $1;



-- name: UpdateWorkspace :one
UPDATE workspaces
SET name = $2, settings = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetAllPermissions :many
SELECT * FROM permissions;

-- name: GetRolePermissions :many
SELECT p.* 
FROM permissions p
JOIN role_permissions rp ON p.id = rp.permission_id
WHERE rp.role_id = $1;

-- name: ClearRolePermissions :exec
DELETE FROM role_permissions WHERE role_id = $1;

-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1 AND is_system = false;

-- name: GetUserByRecoverToken :one
SELECT * FROM users WHERE recover_token LIKE $1 || ':%';
