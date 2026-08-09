-- name: GetSessionByToken :one
SELECT * FROM session
WHERE token = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetWorkspaceByID :one
SELECT * FROM workspaces
WHERE id = $1 LIMIT 1;
