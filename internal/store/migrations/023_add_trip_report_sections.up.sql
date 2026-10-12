-- A section of a trip report's story, like a chapter of a blog post: a
-- heading, text and photos, usually about a stretch of the track. Every
-- report has at least one section.
create table trip_report_sections (
  id uuid primary key default gen_random_uuid(),
  report_id uuid not null references trip_reports(id) on delete cascade,
  position integer not null check (position >= 0),
  -- Empty for a section without a heading, e.g. an introduction.
  title text not null default '',
  -- Plain text, with paragraphs separated by blank lines.
  body text not null default '',
  -- The stretch of the track the section is about, in meters along it.
  start_distance_meters double precision,
  end_distance_meters double precision,
  create_time timestamptz not null default current_timestamp,
  update_time timestamptz not null default current_timestamp,
  check ((start_distance_meters is null) = (end_distance_meters is null)),
  check (start_distance_meters >= 0 and end_distance_meters > start_distance_meters)
);

create index trip_report_sections_report on trip_report_sections (report_id, position);

-- Each report's story becomes its first section, with all its photos.
insert into trip_report_sections (report_id, position, body)
select id, 0, body from trip_reports;

alter table trip_reports drop column body;

-- Photos are ordered by their section, and then by their position.
alter table trip_photos add column section_id uuid references trip_report_sections(id);

update trip_photos p set section_id = s.id
from trip_report_sections s
where s.report_id = p.report_id;

alter table trip_photos alter column section_id set not null;

create index trip_photos_section on trip_photos (section_id);
