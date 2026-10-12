-- A race report: the story of a race, with its result, compared with the
-- race plan. The race's details are copied, so the report stands on its own
-- if the race is deleted.
create table race_reports (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  race_id uuid references races(id) on delete set null,
  activity_id uuid references strava_activities(id) on delete set null,
  title text not null,
  slug text not null check (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
  race_name text not null,
  race_date date not null,
  distance_meters double precision not null default 0,
  elevation_gain_meters double precision not null default 0,
  -- The result: the finish time in seconds, 0 if unknown, and the placement
  -- as free text, e.g. "12th of 240, 3rd M40".
  finish_seconds integer not null default 0,
  placement text not null default '',
  -- The story, in sections of plain text, with paragraphs separated by blank
  -- lines.
  summary text not null default '',
  preparation text not null default '',
  race_day text not null default '',
  nutrition text not null default '',
  went_well text not null default '',
  went_badly text not null default '',
  lessons text not null default '',
  published boolean not null default false,
  -- When the report was first published.
  publish_time timestamptz,
  create_time timestamptz not null default current_timestamp,
  update_time timestamptz not null default current_timestamp,
  unique (user_id, slug)
);

create index race_reports_user_date on race_reports (user_id, race_date desc);
create index race_reports_race on race_reports (race_id);
