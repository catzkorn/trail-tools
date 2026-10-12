-- name: GetProfile :one
select user_id, handle, display_name from public_profiles
where user_id = $1;

-- name: GetProfileByHandle :one
select user_id, handle, display_name from public_profiles
where handle = $1;

-- name: UpsertProfile :one
insert into public_profiles (
  user_id,
  handle,
  display_name
) values (
  $1, $2, $3
)
on conflict (user_id) do update set
  handle = excluded.handle,
  display_name = excluded.display_name,
  update_time = now()
returning user_id, handle, display_name;

-- name: GetUserName :one
select name from passkey_users
where id = $1;

-- name: CreateReport :one
insert into trip_reports (
  user_id,
  activity_id,
  title,
  slug,
  body,
  date,
  timezone,
  sport_type,
  start_time,
  distance_meters,
  moving_time_seconds,
  elapsed_time_seconds,
  elevation_gain_meters,
  track
) values (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
returning *;

-- name: GetReport :one
select * from trip_reports
where id = $1 and user_id = $2;

-- name: GetReportBySlug :one
select * from trip_reports
where user_id = $1 and slug = $2;

-- name: ListReports :many
select
  id, user_id, activity_id, title, slug, body, date, timezone, sport_type, start_time,
  distance_meters, moving_time_seconds, elapsed_time_seconds, elevation_gain_meters,
  (track is not null)::boolean as has_track, published, publish_time, create_time, update_time
from trip_reports
where user_id = $1
  and (sqlc.narg(activity_id)::uuid is null or activity_id = sqlc.narg(activity_id)::uuid)
  and (not sqlc.arg(published_only)::boolean or published)
order by date desc, create_time desc;

-- name: UpdateReport :one
update trip_reports set
  title = coalesce(sqlc.narg(title), title),
  slug = coalesce(sqlc.narg(slug), slug),
  body = coalesce(sqlc.narg(body), body),
  date = coalesce(sqlc.narg(date), date),
  hidden_ends_meters = coalesce(sqlc.narg(hidden_ends_meters), hidden_ends_meters),
  update_time = now()
where id = sqlc.arg(id) and user_id = sqlc.arg(user_id)
returning *;

-- name: SetReportPublished :one
update trip_reports set
  published = sqlc.arg(published),
  publish_time = case when sqlc.arg(published) then coalesce(publish_time, now()) else publish_time end,
  update_time = now()
where id = sqlc.arg(id) and user_id = sqlc.arg(user_id)
returning *;

-- name: ListPhotoObjectKeys :many
select object_key from trip_photos
where report_id = $1;

-- name: DeleteReport :execrows
delete from trip_reports
where id = $1 and user_id = $2;

-- name: ReportExists :one
select exists (select 1 from trip_reports where id = $1 and user_id = $2);

-- name: AddPhoto :one
insert into trip_photos (
  report_id,
  position,
  object_key,
  width,
  height,
  caption,
  lat,
  lng,
  location_source,
  track_distance_meters,
  taken_time,
  strava_photo_id
) values (
  sqlc.arg(report_id),
  (select coalesce(max(position) + 1, 0) from trip_photos where report_id = sqlc.arg(report_id)),
  sqlc.arg(object_key),
  sqlc.arg(width),
  sqlc.arg(height),
  sqlc.arg(caption),
  sqlc.narg(lat),
  sqlc.narg(lng),
  sqlc.narg(location_source),
  sqlc.narg(track_distance_meters),
  sqlc.narg(taken_time),
  sqlc.narg(strava_photo_id)
)
returning *;

-- name: ListPhotos :many
select * from trip_photos
where report_id = any(sqlc.arg(report_ids)::uuid[])
order by report_id, position, create_time;

-- name: GetPhoto :one
select p.* from trip_photos p
join trip_reports r on r.id = p.report_id
where p.id = $1 and r.user_id = $2;

-- name: GetPhotoAccess :one
-- Gets a photo with what's needed to decide who can see it.
select p.object_key, p.create_time, r.user_id, r.published from trip_photos p
join trip_reports r on r.id = p.report_id
where p.id = $1;

-- name: UpdatePhotoCaption :one
update trip_photos p set
  caption = $3
from trip_reports r
where r.id = p.report_id and p.id = $1 and r.user_id = $2
returning p.*;

-- name: UpdatePhotoLocation :one
update trip_photos p set
  lat = sqlc.narg(lat),
  lng = sqlc.narg(lng),
  location_source = sqlc.narg(location_source),
  track_distance_meters = sqlc.narg(track_distance_meters)
from trip_reports r
where r.id = p.report_id and p.id = sqlc.arg(id) and r.user_id = sqlc.arg(user_id)
returning p.*;

-- name: SetPhotoPositions :exec
update trip_photos p set
  position = o.position - 1
from unnest(sqlc.arg(photo_ids)::uuid[]) with ordinality as o(id, position)
where p.id = o.id and p.report_id = sqlc.arg(report_id);

-- name: DeletePhoto :one
delete from trip_photos p
using trip_reports r
where r.id = p.report_id and p.id = $1 and r.user_id = $2
returning p.object_key;

-- name: TouchReport :exec
update trip_reports set
  update_time = now()
where id = $1;
