create table api_tokens (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  name text not null,
  secret_sha256 bytea unique not null,
  create_time timestamptz not null default current_timestamp,
  last_used_time timestamptz
);
