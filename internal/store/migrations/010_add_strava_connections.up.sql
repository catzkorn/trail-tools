create type strava_sync_state as enum (
  'idle',
  'running',
  'rate_limited',
  'failed'
);

create table strava_connections (
  user_id uuid primary key references users(id) on delete cascade,
  athlete_id bigint not null,
  athlete_name text not null,
  access_token text not null,
  refresh_token text not null,
  expiry timestamptz not null,
  sync_state strava_sync_state not null default 'idle',
  sync_message text not null default '',
  last_sync_time timestamptz,
  resume_time timestamptz,
  create_time timestamptz not null default current_timestamp
);
