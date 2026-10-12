-- Workouts have six types. The title and description say what a workout is,
-- so the finer types are folded into their groups.
alter type workout_type rename to workout_type_old;

create type workout_type as enum (
  'base',
  'intensity',
  'cross_training',
  'strength',
  'race',
  'rest'
);

alter table workouts
  alter column type type workout_type using (
    case type::text
      when 'easy' then 'base'
      when 'long_run' then 'base'
      when 'recovery' then 'base'
      when 'vert' then 'base'
      when 'tempo' then 'intensity'
      when 'intervals' then 'intensity'
      when 'hill_repeats' then 'intensity'
      when 'other' then 'cross_training'
      else type::text
    end
  )::workout_type;

drop type workout_type_old;
