-- Whether the user is running the race, rather than considering it. Races on
-- the calendar are shown on it and can have race plans.
alter table races
  add column on_calendar boolean not null default false;

-- Races with race plans or that training plans work towards are being run.
update races r set on_calendar = true
where exists (select 1 from race_plans p where p.race_id = r.id)
  or exists (select 1 from training_plans t where r.id = any(t.goal_race_ids));
