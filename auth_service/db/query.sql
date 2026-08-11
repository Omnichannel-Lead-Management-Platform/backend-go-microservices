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
INSERT INTO users (workspace_id, role, name, email, password, "emailVerified")
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
