create type routing_profile as enum (
  'trail',
  'shortest',
  'straight',
  -- Prefer the paths in the user's personal heatmap.
  'heatmap'
);

create type route_source as enum (
  'waypoints',
  'gpx',
  'activity'
);

create table routes (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  name text not null,
  description text not null,
  profile routing_profile not null,
  source route_source not null,
  -- The route geometry and computed statistics, see internal/routes.
  waypoints jsonb not null,
  legs jsonb not null,
  stats jsonb not null,
  create_time timestamptz not null default current_timestamp,
  update_time timestamptz not null default current_timestamp
);

create index routes_user on routes (user_id, update_time desc);
