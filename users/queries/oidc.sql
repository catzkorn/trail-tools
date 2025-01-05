-- name: CreateOIDCUser :one
insert into oidc_users (subject) values ($1) returning *;

-- name: GetOIDCUser :one
select * from oidc_users where subject = $1;
