-- How far around the start and finish of a trip report's track is hidden
-- on its public page, e.g. so that it doesn't show where the author lives.
alter table trip_reports
  add column hidden_ends_meters integer not null default 500
    check (hidden_ends_meters between 0 and 2000);
