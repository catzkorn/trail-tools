-- When the sync state last changed. A running sync updates its state as it
-- makes progress, so a running sync that hasn't updated it for a while was
-- interrupted.
alter table strava_connections
  add column sync_state_time timestamptz not null default current_timestamp;

-- Activities are listed newest first. While a listing is unfinished, the
-- activities that started before this time remain to be listed.
alter table strava_connections
  add column list_before timestamptz;
