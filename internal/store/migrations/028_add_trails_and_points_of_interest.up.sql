-- Trails and points of interest are found in the imported OpenStreetMap data
-- rather than with the Overpass API. Regions imported before have no names or
-- points of interest until they are imported again.

-- The OpenStreetMap way an edge is a section of, null for edges imported
-- before ways were recorded, and the tags that describe it.
alter table routing_edges
  add column way_id bigint,
  add column name text not null default '',
  add column surface text not null default '';

-- Finds the edges of the trails found near a point, to return whole ways. The
-- highway types are those FindTrails returns.
create index routing_edges_trail_way on routing_edges (way_id)
where highway in ('path', 'footway', 'track', 'bridleway', 'steps', 'cycleway');

-- OpenStreetMap nodes and ways useful for trail running, such as peaks,
-- water and car parks, imported with the routing graph.
create table points_of_interest (
  region_id smallint not null references routing_regions (id) on delete cascade,
  -- The OpenStreetMap ID of the node or way.
  osm_id bigint not null,
  -- Whether it is a way, located at the center of its bounding box.
  is_way boolean not null,
  -- peak, water, parking, viewpoint, shelter, toilets, saddle or trailhead.
  kind text not null,
  name text not null,
  location geometry(Point, 4326) not null,
  -- From the ele tag, or the elevation model if it has none. Null if
  -- unknown.
  elevation_meters real
);

create index points_of_interest_location on points_of_interest using gist (location);
create index points_of_interest_region on points_of_interest (region_id);
