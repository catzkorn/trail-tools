-- The photo shown on top of a trip report, in lists of reports and in link
-- previews. Null to use the first photo. It must be one of the report's
-- photos, and deleting it goes back to the first photo.
alter table trip_photos
  add constraint trip_photos_report_id_id_key unique (report_id, id);

alter table trip_reports
  add column cover_photo_id uuid,
  add constraint trip_reports_cover_photo_fkey
    foreign key (id, cover_photo_id) references trip_photos (report_id, id)
    on delete set null (cover_photo_id);
