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

-- ListAPITokens lists the user's API tokens, excluding OAuth access tokens.
-- name: ListAPITokens :many
select * from api_tokens
where user_id = $1 and oauth_grant_id is null
order by create_time desc;

-- name: DeleteAPIToken :execrows
delete from api_tokens
where id = $1 and user_id = $2 and oauth_grant_id is null;

-- name: UseAPIToken :one
update api_tokens
set last_used_time = now()
where secret_sha256 = $1 and (expire_time is null or expire_time > now())
returning user_id;
