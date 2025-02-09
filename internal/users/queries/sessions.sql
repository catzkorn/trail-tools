-- name: CreateSession :one
insert into sessions (user_id, expiry) values ($1, $2) returning id;

-- name: GetSessionUser :one
select
  user_id,
  case 
    when exists(select 1 from oidc_users where sessions.user_id=oidc_users.id) then 'oidc'
    when exists(select 1 from web_authn_users where sessions.user_id=web_authn_users.id) then 'webauthn'
  end::text as type
from sessions where sessions.id = $1 and sessions.expiry > now();

-- name: DeleteSession :exec
delete from sessions where id = $1;
