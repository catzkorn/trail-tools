-- name: CreateLactateTest :one
insert into lactate_tests (
  user_id,
  date,
  notes,
  resting_lactate_mmol_per_liter,
  stage_heart_rates_bpm,
  stage_lactates_mmol_per_liter
) values (
  $1, $2, $3, $4, $5, $6
)
returning *;

-- name: GetLactateTest :one
select * from lactate_tests
where id = $1 and user_id = $2;

-- name: ListLactateTests :many
select * from lactate_tests
where user_id = $1
order by date desc, create_time desc;

-- name: UpdateLactateTest :one
update lactate_tests set
  date = $3,
  notes = $4,
  resting_lactate_mmol_per_liter = $5,
  stage_heart_rates_bpm = $6,
  stage_lactates_mmol_per_liter = $7,
  update_time = now()
where id = $1 and user_id = $2
returning *;

-- name: DeleteLactateTest :execrows
delete from lactate_tests
where id = $1 and user_id = $2;
