create table athletes (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id),
  create_time timestamptz not null default current_timestamp,
  name text not null
);

create table activities (
  id uuid primary key default gen_random_uuid(),
  athlete_id uuid not null references athletes(id),
  create_time timestamptz not null default current_timestamp,
  name text not null
);

create table blood_lactate_measures (
  id uuid primary key default gen_random_uuid(),
  activity_id uuid not null references activities(id),
  create_time timestamptz not null default current_timestamp,
  mmol_per_liter numeric not null,
  heart_rate_bpm integer not null
);
