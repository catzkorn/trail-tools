-- name: GetUser :one
INSERT INTO users (
  oidc_subject
) VALUES (
  $1
)
ON CONFLICT (oidc_subject) DO NOTHING
RETURNING *;
