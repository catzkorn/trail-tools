-- name: CreateTrainingPlan :one
insert into training_plans (
  user_id,
  name,
  description,
  start_date,
  end_date,
  goal_race_ids
) values (
  $1, $2, $3, $4, $5, $6
)
returning *;

-- name: GetTrainingPlan :one
select * from training_plans
where id = $1 and user_id = $2;

-- name: ListTrainingPlans :many
select * from training_plans
where user_id = $1
order by start_date desc;

-- name: UpdateTrainingPlan :one
update training_plans set
  name = $3,
  description = $4,
  start_date = $5,
  end_date = $6,
  goal_race_ids = $7,
  update_time = now()
where id = $1 and user_id = $2
returning *;

-- name: DeleteTrainingPlan :execrows
delete from training_plans
where id = $1 and user_id = $2;

-- name: UpsertPlanPeriod :one
insert into plan_periods (
  plan_id,
  granularity,
  start_date,
  title,
  notes,
  target_distance_meters,
  target_elevation_gain_meters,
  target_duration_seconds
) values (
  $1, $2, $3, $4, $5, $6, $7, $8
)
on conflict (plan_id, granularity, start_date) do update set
  title = excluded.title,
  notes = excluded.notes,
  target_distance_meters = excluded.target_distance_meters,
  target_elevation_gain_meters = excluded.target_elevation_gain_meters,
  target_duration_seconds = excluded.target_duration_seconds
returning *;

-- name: ListPlanPeriods :many
select * from plan_periods
where plan_id = $1
order by start_date, granularity;

-- name: DeletePlanPeriods :execrows
delete from plan_periods
where plan_id = $1 and id = any(sqlc.arg(ids)::uuid[]);

-- name: CreateWorkout :one
insert into workouts (
  plan_id,
  date,
  type,
  title,
  description,
  target_distance_meters,
  target_elevation_gain_meters,
  target_duration_seconds,
  target_terrain,
  route_id,
  completed_activity_id
) values (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
returning *;

-- name: ListWorkouts :many
select * from workouts
where plan_id = $1
  and (sqlc.narg(start_date)::date is null or date >= sqlc.narg(start_date)::date)
  and (sqlc.narg(end_date)::date is null or date <= sqlc.narg(end_date)::date)
order by date, create_time;

-- name: GetWorkout :one
select w.* from workouts w
join training_plans p on p.id = w.plan_id
where w.id = $1 and p.user_id = $2;

-- name: UpdateWorkout :one
update workouts set
  date = $2,
  type = $3,
  title = $4,
  description = $5,
  target_distance_meters = $6,
  target_elevation_gain_meters = $7,
  target_duration_seconds = $8,
  target_terrain = $9,
  route_id = $10,
  completed_activity_id = $11
where id = $1
returning *;

-- name: DeleteWorkouts :execrows
delete from workouts
where plan_id = $1 and id = any(sqlc.arg(ids)::uuid[]);

-- name: RouteExists :one
select exists(select 1 from routes where id = $1 and user_id = $2);

-- name: ActivityExists :one
select exists(select 1 from strava_activities where id = $1 and user_id = $2);

-- name: CountRaces :one
select count(*) from races
where user_id = $1 and id = any(sqlc.arg(ids)::uuid[]);
