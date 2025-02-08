-- name: CreateOIDCUser :one
insert into oidc_users (subject) values ($1) on conflict (subject) do update set subject=excluded.subject returning *;

-- name: GetOIDCUser :one
select * from oidc_users where id = $1;
