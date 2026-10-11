-- OAuth clients, such as claude.ai custom connectors, act on behalf of users
-- with OAuth grants. Clients register dynamically without being stored: their
-- client ID encodes their registration.

-- An authorization code, issued when a user approves an OAuth client. Codes
-- are single use and short lived.
create table oauth_authorization_codes (
  code_sha256 bytea primary key,
  user_id uuid not null references users(id) on delete cascade,
  client_id text not null,
  client_name text not null,
  redirect_uri text not null,
  -- The S256 PKCE code challenge.
  code_challenge text not null,
  expire_time timestamptz not null,
  create_time timestamptz not null default current_timestamp
);

-- A user's authorization of an OAuth client. Deleting it revokes the
-- client's tokens.
create table oauth_grants (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  client_id text not null,
  client_name text not null,
  redirect_uri text not null,
  -- The current refresh token. Refresh tokens are rotated on every use.
  refresh_token_sha256 bytea unique not null,
  refresh_expire_time timestamptz not null,
  create_time timestamptz not null default current_timestamp
);

create index on oauth_grants (user_id);

-- OAuth access tokens are API tokens belonging to a grant, and expire.
alter table api_tokens
  add column oauth_grant_id uuid references oauth_grants(id) on delete cascade,
  add column expire_time timestamptz;

create index on api_tokens (oauth_grant_id);
