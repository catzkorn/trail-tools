create table training_plans (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  name text not null,
  description text not null,
  start_date date not null,
  end_date date not null,
  goal_race_ids uuid[] not null,
  create_time timestamptz not null default current_timestamp,
  update_time timestamptz not null default current_timestamp,
  check (end_date >= start_date)
);

create type period_granularity as enum (
  'month',
  'week'
);

create table plan_periods (
  id uuid primary key default gen_random_uuid(),
  plan_id uuid not null references training_plans(id) on delete cascade,
  granularity period_granularity not null,
  start_date date not null,
  title text not null,
  notes text not null,
  target_distance_meters double precision not null,
  target_elevation_gain_meters double precision not null,
  target_duration_seconds integer not null,
  unique (plan_id, granularity, start_date)
);

create type workout_type as enum (
  'easy',
  'long_run',
  'tempo',
  'intervals',
  'hill_repeats',
  'vert',
  'recovery',
  'rest',
  'cross_training',
  'strength',
  'race',
  'other'
);

create type terrain as enum (
  'paved',
  'gravel',
  'trail',
  'technical'
);

create table workouts (
  id uuid primary key default gen_random_uuid(),
  plan_id uuid not null references training_plans(id) on delete cascade,
  date date not null,
  type workout_type not null,
  title text not null,
  description text not null,
  target_distance_meters double precision not null,
  target_elevation_gain_meters double precision not null,
  target_duration_seconds integer not null,
  target_terrain terrain,
  route_id uuid references routes(id) on delete set null,
  completed_activity_id uuid references strava_activities(id) on delete set null,
  create_time timestamptz not null default current_timestamp
);

create index workouts_plan_date on workouts (plan_id, date);
