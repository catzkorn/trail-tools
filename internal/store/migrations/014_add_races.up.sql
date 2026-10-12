create type race_priority as enum (
  'a',
  'b',
  'c'
);

create table races (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  name text not null,
  date date not null,
  start_time text not null,
  location text not null,
  website_url text not null,
  distance_meters double precision not null,
  elevation_gain_meters double precision not null,
  cutoff_seconds integer not null,
  priority race_priority,
  course_route_id uuid references routes(id) on delete set null,
  notes text not null,
  create_time timestamptz not null default current_timestamp,
  update_time timestamptz not null default current_timestamp
);
