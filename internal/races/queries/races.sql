-- name: CreateRace :one
insert into races (
  user_id,
  name,
  date,
  start_time,
  location,
  website_url,
  distance_meters,
  elevation_gain_meters,
  cutoff_seconds,
  priority,
  course_route_id,
  notes
) values (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
returning *;

-- name: GetRace :one
select * from races
where id = $1 and user_id = $2;

-- name: ListRaces :many
select * from races
where user_id = $1
order by date, name;

-- name: UpdateRace :one
update races set
  name = $3,
  date = $4,
  start_time = $5,
  location = $6,
  website_url = $7,
  distance_meters = $8,
  elevation_gain_meters = $9,
  cutoff_seconds = $10,
  priority = $11,
  course_route_id = $12,
  notes = $13,
  update_time = now()
where id = $1 and user_id = $2
returning *;

-- name: DeleteRace :execrows
delete from races
where id = $1 and user_id = $2;

-- name: RemoveGoalRace :exec
update training_plans set
  goal_race_ids = array_remove(goal_race_ids, sqlc.arg(race_id)::uuid)
where user_id = $1;

-- name: CreateRacePlan :one
insert into race_plans (
  user_id,
  race_id,
  sections
) values (
  $1, $2, $3
)
returning *;

-- name: GetRacePlan :one
select * from race_plans
where id = $1 and user_id = $2;

-- name: GetRacePlanByShareToken :one
select * from race_plans
where share_token = $1;

-- name: ListRacePlans :many
select * from race_plans
where user_id = $1;

-- name: UpdateRacePlanSections :one
update race_plans set
  sections = $3,
  update_time = now()
where id = $1 and user_id = $2
returning *;

-- name: SetRacePlanShareToken :one
update race_plans set
  share_token = $3
where id = $1 and user_id = $2
returning *;

-- name: DeleteRacePlan :execrows
delete from race_plans
where id = $1 and user_id = $2;

-- name: GetRaceByID :one
select * from races
where id = $1;

-- name: SetRacePlanRaceDay :one
update race_plans set
  race_day = $2
where id = $1
returning *;
