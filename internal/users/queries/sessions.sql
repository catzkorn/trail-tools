-- name: CreateSession :one
insert into sessions (user_id, expiry) values ($1, $2) returning id;

-- name: GetSessionUser :one
select p.*, u.is_admin, u.avatar_key from sessions s
join passkey_users p on p.id = s.user_id
join users u on u.id = s.user_id
where s.id = $1 and s.expiry > now();

-- name: DeleteSession :exec
delete from sessions where id = $1;
