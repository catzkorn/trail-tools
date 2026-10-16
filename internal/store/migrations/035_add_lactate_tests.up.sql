-- Blood lactate tests, which find the heart rate of the user's aerobic
-- threshold. Each stage is a heart rate held for a few minutes and the
-- lactate measured at its end. The stages are stored as two arrays of the
-- same length, as they're always read and written together.
create table lactate_tests (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  date date not null,
  notes text not null default '',
  -- Null if not measured.
  resting_lactate_mmol_per_liter double precision,
  stage_heart_rates_bpm integer[] not null default '{}',
  stage_lactates_mmol_per_liter double precision[] not null default '{}',
  create_time timestamptz not null default current_timestamp,
  update_time timestamptz not null default current_timestamp,
  check (cardinality(stage_heart_rates_bpm) = cardinality(stage_lactates_mmol_per_liter))
);

create index lactate_tests_user_date on lactate_tests (user_id, date desc);
