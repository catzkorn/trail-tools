-- Users sign in with passkeys only. Delete the users who signed in with
-- OIDC, along with their sessions and data. Their athletes, activities and
-- measures don't cascade, so delete those first.
delete from blood_lactate_measures where activity_id in (
  select activities.id from activities
  join athletes on athletes.id = activities.athlete_id
  where athletes.user_id in (select id from oidc_users)
);
delete from activities where athlete_id in (
  select id from athletes where user_id in (select id from oidc_users)
);
delete from athletes where user_id in (select id from oidc_users);
delete from users where id in (select id from oidc_users);
drop table oidc_users;
