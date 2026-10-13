-- The tags of a trip report, for readers to find reports, e.g. good for
-- beginners or technical. The names are those of trips.v1.TripTag, without
-- the prefix and in lowercase.
alter table trip_reports
  add column tags text[] not null default '{}'
    check (tags <@ array[
      'beginner_friendly', 'moderate', 'challenging',
      'runnable', 'technical', 'scrambling', 'snow',
      'loop', 'out_and_back', 'point_to_point'
    ]::text[]);
