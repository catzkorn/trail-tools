-- A copy of the race activity's track, downsampled, as a protobuf encoded
-- activities.v1.ActivityStreams message, which segments are distances along.
-- It's copied when the activity is set, so segments stay in place if the
-- activity changes on Strava. Null without an activity or positions.
alter table race_reports add column track bytea;

-- Stretches of the race's track the runner comments on, e.g. "The climb to
-- the ridge", shown highlighted on its map and elevation profile.
create table race_report_segments (
  id uuid primary key default gen_random_uuid(),
  report_id uuid not null references race_reports(id) on delete cascade,
  position integer not null check (position >= 0),
  title text not null default '',
  body text not null default '',
  -- Where the stretch starts and ends, in meters along the report's track.
  start_distance_meters double precision not null check (start_distance_meters >= 0),
  end_distance_meters double precision not null,
  create_time timestamptz not null default current_timestamp,
  check (end_distance_meters > start_distance_meters)
);

create index race_report_segments_report on race_report_segments (report_id, position);
