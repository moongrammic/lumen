-- name: CreateUser :one
INSERT INTO users (
  username,
  email,
  password_hash
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT id, username, email, avatar_url, about_me, created_at FROM users
WHERE id = $1 LIMIT 1;

-- name: CreateWorkspace :one
INSERT INTO workspaces (
  name,
  owner_id
) VALUES (
  $1, $2
)
RETURNING *;

-- name: AddWorkspaceMember :one
INSERT INTO members (
  user_id,
  workspace_id
) VALUES (
  $1, $2
)
RETURNING *;

-- name: ListUserWorkspaces :many
SELECT w.* FROM workspaces w
JOIN members m ON w.id = m.workspace_id
WHERE m.user_id = $1;

-- name: IsWorkspaceMember :one
SELECT EXISTS(
  SELECT 1 FROM members WHERE user_id = $1 AND workspace_id = $2
);