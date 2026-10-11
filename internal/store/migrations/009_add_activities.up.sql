-- Activities are synced from Strava, see internal/activities.
create table strava_activities (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  strava_id bigint not null,
  name text not null,
  description text not null default '',
  sport_type text not null,
  start_time timestamptz not null,
  start_date_local date not null,
  timezone text not null,
  distance_meters double precision not null,
  moving_time_seconds integer not null,
  elapsed_time_seconds integer not null,
  elevation_gain_meters double precision not null,
  average_speed_mps double precision not null,
  max_speed_mps double precision not null,
  average_heartrate_bpm double precision not null,
  max_heartrate_bpm double precision not null,
  average_cadence_spm double precision not null,
  summary_polyline text not null,
  device_name text not null default '',
  relative_effort integer not null,
  -- The raw Strava API responses, kept so new fields can be backfilled
  -- without re-syncing.
  summary_json jsonb not null,
  detail_json jsonb,
  -- The bounding box of the GPS track, null if the activity has no GPS data.
  min_lat double precision,
  min_lng double precision,
  max_lat double precision,
  max_lng double precision,
  streams_synced boolean not null default false,
  create_time timestamptz not null default current_timestamp,
  update_time timestamptz not null default current_timestamp,
  unique (user_id, strava_id)
);

create index strava_activities_user_date on strava_activities (user_id, start_date_local desc);

create table strava_activity_streams (
  activity_id uuid primary key references strava_activities(id) on delete cascade,
  -- A protobuf encoded activities.v1.ActivityStreams message.
  streams bytea not null
);
