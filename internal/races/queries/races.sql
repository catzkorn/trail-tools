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
