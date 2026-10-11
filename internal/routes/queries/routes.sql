-- name: CreateRoute :one
insert into routes (
  user_id,
  name,
  description,
  profile,
  source,
  waypoints,
  legs,
  stats
) values (
  $1, $2, $3, $4, $5, $6, $7, $8
)
returning *;

-- name: GetRoute :one
select * from routes
where id = $1 and user_id = $2;

-- name: ListRoutes :many
select id, user_id, name, description, profile, source, waypoints, stats, create_time, update_time
from routes
where user_id = $1
order by update_time desc;

-- name: UpdateRoute :one
update routes set
  name = $3,
  description = $4,
  profile = $5,
  waypoints = $6,
  legs = $7,
  stats = $8,
  update_time = now()
where id = $1 and user_id = $2
returning *;

-- name: DeleteRoute :execrows
delete from routes
where id = $1 and user_id = $2;
