-- The GPS track of an activity from its streams, simplified to a couple of
-- meters and encoded as a Google polyline, for drawing and searching tracks
-- without reading the whole streams. Empty if the streams have no positions,
-- and null until it's computed: when the streams are synced, or the first
-- time the track is listed for older streams.
alter table strava_activity_streams add column track_polyline text;
