-- The public profile of a user, under which published trip reports are
-- listed at /u/<handle>.
create table public_profiles (
  user_id uuid primary key references users(id) on delete cascade,
  handle text not null unique check (
    handle ~ '^[a-z0-9]([a-z0-9-]*[a-z0-9])?$' and length(handle) between 3 and 30
  ),
  display_name text not null,
  create_time timestamptz not null default current_timestamp,
  update_time timestamptz not null default current_timestamp
);

-- A trip report, published at /u/<handle>/<slug>. The statistics and track
-- are copied from the activity when the report is created, so the report
-- doesn't change when the activity does.
create table trip_reports (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  activity_id uuid references strava_activities(id) on delete set null,
  title text not null,
  slug text not null check (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
  -- Plain text, with paragraphs separated by blank lines.
  body text not null,
  date date not null,
  timezone text not null,
  sport_type text not null,
  start_time timestamptz,
  distance_meters double precision not null,
  moving_time_seconds integer not null,
  elapsed_time_seconds integer not null,
  elevation_gain_meters double precision not null,
  -- A protobuf encoded activities.v1.ActivityStreams message with the time,
  -- distance, position and altitude of the track, downsampled. Null if the
  -- report has no track.
  track bytea,
  published boolean not null default false,
  -- When the report was first published.
  publish_time timestamptz,
  create_time timestamptz not null default current_timestamp,
  update_time timestamptz not null default current_timestamp,
  unique (user_id, slug)
);

create index trip_reports_user_date on trip_reports (user_id, date desc);
create index trip_reports_activity on trip_reports (activity_id);

create type photo_location_source as enum (
  'gps',
  'track_time',
  'manual'
);

-- A photo of a trip report. The images are stored in the blob store under
-- <object_key>/large.jpg and <object_key>/thumb.jpg.
create table trip_photos (
  id uuid primary key default gen_random_uuid(),
  report_id uuid not null references trip_reports(id) on delete cascade,
  position integer not null check (position >= 0),
  object_key text not null unique,
  -- The size of the large image.
  width integer not null check (width > 0),
  height integer not null check (height > 0),
  caption text not null,
  lat double precision,
  lng double precision,
  location_source photo_location_source,
  -- The distance along the report's track of the point nearest to the
  -- location.
  track_distance_meters double precision,
  taken_time timestamptz,
  -- Strava's ID of the photo, if it was imported from the activity on
  -- Strava, so it isn't imported twice.
  strava_photo_id text,
  create_time timestamptz not null default current_timestamp,
  unique (report_id, strava_photo_id),
  check ((lat is null) = (lng is null)),
  check ((lat is null) = (location_source is null))
);

create index trip_photos_report on trip_photos (report_id, position);
