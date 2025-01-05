-- name: AddAthlete :one
insert into athletes (
  name,
  user_id
) values (
  $1,
  $2
)
returning *;

-- name: DeleteAthlete :one
delete from athletes
where id = $1
returning *;

-- name: AddActivity :one
insert into activities (
  name,
  athlete_id
) values (
  $1,
  $2
)
returning *;

-- name: AddMeasure :one
insert into blood_lactate_measures (
  activity_id,
  mmol_per_liter,
  heart_rate_bpm
) values (
  $1,
  $2,
  $3
)
returning *;
