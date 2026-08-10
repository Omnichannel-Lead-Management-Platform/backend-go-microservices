-- name: GetSessionByToken :one
SELECT * FROM session
WHERE token = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

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
