create type routing_import_state as enum (
  'queued',
  'running',
  'done',
  'failed'
);

-- Imports of OpenStreetMap data into the routing graph, started from the map
-- data page and run in the background or as a Cloud Run job.
create table routing_imports (
  id uuid primary key default gen_random_uuid(),
  -- The region to import into, replacing any earlier import of it.
  region text not null,
  -- The area to import.
  south double precision not null,
  west double precision not null,
  north double precision not null,
  east double precision not null,
  -- The OpenStreetMap extracts the area is read from, more than one if it
  -- crosses a border. The URLs are in the same order as the names.
  extract_names text[] not null,
  extract_urls text[] not null,
  state routing_import_state not null default 'queued',
  -- Why the import failed.
  message text not null default '',
  edge_count integer not null default 0,
  created_by uuid references users(id) on delete set null,
  create_time timestamptz not null default current_timestamp,
  start_time timestamptz,
  finish_time timestamptz
);

create index routing_imports_create_time on routing_imports (create_time desc);

-- At most one import is queued or running at a time, as an import can use a
-- lot of memory.
create unique index routing_imports_active on routing_imports ((true))
where state in ('queued', 'running');
