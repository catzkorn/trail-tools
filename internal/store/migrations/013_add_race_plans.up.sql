create table race_plans (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  race_id uuid not null references races(id) on delete cascade,
  -- A JSON encoded races.v1.RacePlan message containing the plan sections.
  sections jsonb not null,
  share_token text unique,
  create_time timestamptz not null default current_timestamp,
  update_time timestamptz not null default current_timestamp
);
