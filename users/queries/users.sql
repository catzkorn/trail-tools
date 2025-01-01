-- name: GetUser :one
INSERT INTO users (
  oidc_subject
) VALUES (
  $1
)
-- Note: we don't actually need to update anything, but if we don't,
-- the query doesn't return the row.
ON CONFLICT (oidc_subject) DO UPDATE set oidc_subject = EXCLUDED.oidc_subject
RETURNING *;
