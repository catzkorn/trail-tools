-- Places the user's tracks and photos are hidden around on public pages,
-- like their home. They replace hiding the start and finish of each trip
-- report, as trips and races often start and finish elsewhere.
create table privacy_zones (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  -- A name for the user, e.g. "Home".
  name text not null default '',
  lat double precision not null check (lat between -90 and 90),
  lng double precision not null check (lng between -180 and 180),
  radius_meters integer not null check (radius_meters between 100 and 5000),
  create_time timestamptz not null default current_timestamp
);

create index privacy_zones_user on privacy_zones (user_id);

alter table trip_reports drop column hidden_ends_meters;
