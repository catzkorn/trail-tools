-- What happened on race day, as recorded by the runner or crew.
alter table race_plans
  add column race_day_livetrack_url text not null default '',
  -- Null if the start time hasn't been recorded.
  add column race_day_start_time timestamptz;

-- The recorded times at a checkpoint on race day. The times are kept when the
-- plan's checkpoints are edited, and deleted with their checkpoint.
create table race_plan_splits (
  plan_id uuid not null,
  checkpoint_position integer not null,
  arrival_time timestamptz,
  departure_time timestamptz,
  primary key (plan_id, checkpoint_position),
  foreign key (plan_id, checkpoint_position) references race_plan_checkpoints (plan_id, position) on delete cascade,
  check (arrival_time is not null or departure_time is not null),
  check (departure_time >= arrival_time)
);
