CREATE TABLE athlete (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  name TEXT NOT NULL
);

CREATE TABLE activity (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  name TEXT NOT NULL,
  athlete_id UUID NOT NULL REFERENCES athlete(id)
);

CREATE TABLE blood_lactate_measure (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  activity_id UUID NOT NULL REFERENCES activity(id),
  mmol_per_liter NUMERIC NOT NULL,
  heart_rate_bpm INTEGER NOT NULL
);

