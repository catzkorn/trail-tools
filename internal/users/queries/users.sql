-- name: GetUser :one
select * from users where id = $1;

-- name: GetAvatarKeyForUpdate :one
select avatar_key from users where id = $1 for update;

-- name: SetAvatarKey :exec
update users set avatar_key = $2 where id = $1;
