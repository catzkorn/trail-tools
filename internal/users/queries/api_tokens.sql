-- name: CreateAPIToken :one
insert into api_tokens (
  user_id,
  name,
  secret_sha256
) values (
  $1,
  $2,
  $3
)
returning *;

-- name: ListAPITokens :many
select * from api_tokens
where user_id = $1
order by create_time desc;

-- name: DeleteAPIToken :execrows
delete from api_tokens
where id = $1 and user_id = $2;

-- name: UseAPIToken :one
update api_tokens
set last_used_time = now()
where secret_sha256 = $1
returning user_id;
