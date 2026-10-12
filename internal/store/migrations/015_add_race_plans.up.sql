-- A race plan. The sections of the plan with a single value (goals, pacing,
-- nutrition, crew and mental strategy) are columns prefixed with the section
-- name, lists are child tables ordered by position.
create table race_plans (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  race_id uuid not null references races(id) on delete cascade,
  share_token text unique,
  goals_a_finish_seconds integer not null,
  goals_b_finish_seconds integer not null,
  goals_c_finish_seconds integer not null,
  goals_notes text not null,
  pacing_strategy text not null,
  pacing_hike_grade_percent double precision not null,
  pacing_heart_rate_cap_bpm integer not null,
  pacing_notes text not null,
  nutrition_carbs_grams_per_hour double precision not null,
  nutrition_fluid_ml_per_hour double precision not null,
  nutrition_sodium_mg_per_hour double precision not null,
  nutrition_notes text not null,
  crew_has_crew boolean not null,
  crew_members text not null,
  crew_notes text not null,
  mental_mantras text[] not null,
  mental_notes text not null,
  notes text not null,
  create_time timestamptz not null default current_timestamp,
  update_time timestamptz not null default current_timestamp
);

create index race_plans_user on race_plans (user_id);
create index race_plans_race on race_plans (race_id);

-- The checkpoints of a plan in course order, each with the segment leading
-- to it. Checkpoints are updated in place by position, so that race day
-- data referencing them survives edits to the plan.
create table race_plan_checkpoints (
  plan_id uuid not null references race_plans(id) on delete cascade,
  position integer not null check (position >= 0),
  name text not null,
  distance_meters double precision not null check (distance_meters >= 0),
  elevation_gain_meters double precision not null,
  elevation_loss_meters double precision not null,
  cutoff_seconds integer not null check (cutoff_seconds >= 0),
  crew_access boolean not null,
  drop_bag boolean not null,
  aid text not null,
  planned_segment_seconds integer not null check (planned_segment_seconds >= 0),
  planned_stop_seconds integer not null check (planned_stop_seconds >= 0),
  segment_strategy text not null,
  tasks text not null,
  crew_instructions text not null,
  drop_bag_contents text not null,
  primary key (plan_id, position)
);

-- The fuel to consume on the segment leading to a checkpoint.
create table race_plan_intakes (
  plan_id uuid not null,
  checkpoint_position integer not null,
  position integer not null check (position >= 0),
  product text not null,
  quantity double precision not null check (quantity >= 0),
  primary key (plan_id, checkpoint_position, position),
  foreign key (plan_id, checkpoint_position) references race_plan_checkpoints (plan_id, position) on delete cascade
);

create table race_plan_nutrition_products (
  plan_id uuid not null references race_plans(id) on delete cascade,
  position integer not null check (position >= 0),
  name text not null,
  carbs_grams double precision not null,
  sodium_mg double precision not null,
  fluid_ml double precision not null,
  caffeine_mg double precision not null,
  primary key (plan_id, position)
);

create table race_plan_gear (
  plan_id uuid not null references race_plans(id) on delete cascade,
  position integer not null check (position >= 0),
  name text not null,
  mandatory boolean not null,
  location text not null,
  packed boolean not null,
  notes text not null,
  primary key (plan_id, position)
);

create type contingency_scenario as enum (
  'gi_distress',
  'behind_schedule',
  'low_energy',
  'injury',
  'weather',
  'gear_failure',
  'low_motivation',
  'navigation',
  'other'
);

create table race_plan_contingencies (
  plan_id uuid not null references race_plans(id) on delete cascade,
  position integer not null check (position >= 0),
  scenario contingency_scenario not null,
  trigger text not null,
  response text not null,
  primary key (plan_id, position)
);

create table race_plan_race_morning (
  plan_id uuid not null references race_plans(id) on delete cascade,
  position integer not null check (position >= 0),
  -- The local time as entered, e.g. "04:30".
  time text not null,
  description text not null,
  primary key (plan_id, position)
);
