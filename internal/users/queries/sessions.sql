-- name: CreateSession :one
insert into sessions (user_id, expiry) values ($1, $2) returning id;

-- name: GetSessionUser :one
select user_id from sessions where sessions.id = $1 and sessions.expiry > now();

-- name: DeleteSession :exec
delete from sessions where id = $1;
