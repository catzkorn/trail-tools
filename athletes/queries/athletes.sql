-- name: AddAthlete :one
INSERT INTO athletes (
  name,
  user_id
) VALUES (
  $1,
  $2
)
RETURNING *;

-- name: DeleteAthlete :one
DELETE FROM athletes
WHERE id = $1
RETURNING *;

-- name: AddActivity :one
INSERT INTO activities (
  name,
  athlete_id
) VALUES (
  $1,
  $2
)
RETURNING *;

-- name: AddMeasure :one
INSERT INTO blood_lactate_measures (
  activity_id,
  mmol_per_liter,
  heart_rate_bpm
) VALUES (
  $1,
  $2,
  $3
)
RETURNING *;
