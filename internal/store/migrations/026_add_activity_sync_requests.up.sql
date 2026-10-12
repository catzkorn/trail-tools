-- Activities a user asked to sync, e.g. by opening them while the backlog of
-- activities was rate limited, are synced before the rest of the backlog.
alter table strava_activities
  add column sync_requested boolean not null default false;
