-- A race report's result can be that the runner did not finish (DNF): where
-- and when they stopped instead of a finish time. The stop distance is along
-- the course in meters, the stop checkpoint the name of the checkpoint or
-- place, and the stop time the elapsed time in seconds; 0 or empty if
-- unknown.
alter table race_reports
  add column did_not_finish boolean not null default false,
  add column stop_distance_meters double precision not null default 0 check (stop_distance_meters >= 0),
  add column stop_checkpoint text not null default '',
  add column stop_seconds integer not null default 0 check (stop_seconds >= 0);
