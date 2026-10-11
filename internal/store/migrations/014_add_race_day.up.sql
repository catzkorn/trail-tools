-- A JSON encoded races.v1.RaceDay message with the race day progress.
alter table race_plans add column race_day jsonb;
