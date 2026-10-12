-- An event groups races that belong together, e.g. a 50 mile race on the
-- Saturday and a 50K on the Sunday, or the races of a festival weekend.
create table race_events (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  name text not null,
  location text not null,
  website_url text not null,
  notes text not null,
  create_time timestamptz not null default current_timestamp,
  update_time timestamptz not null default current_timestamp
);

create index race_events_user_id_idx on race_events (user_id);

-- Deleting an event keeps its races.
alter table races
  add column event_id uuid references race_events(id) on delete set null;

create index races_event_id_idx on races (event_id);
