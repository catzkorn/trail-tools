-- name: AddAthlete :one
INSERT INTO athlete (
  name
) VALUES (
  $1
)
RETURNING *;

-- name: DeleteAthlete :one
DELETE FROM athlete
WHERE id = $1
RETURNING *;

-- name: AddActivity :one
INSERT INTO activity (
  name,
  athlete_id
) VALUES (
  $1,
  $2
)
RETURNING *;

-- name: AddMeasure :one
INSERT INTO blood_lactate_measure (
  activity_id,
  mmol_per_liter,
  heart_rate_bpm
) VALUES (
  $1,
  $2,
  $3
)
RETURNING *;
