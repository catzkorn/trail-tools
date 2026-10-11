-- Routing on OpenStreetMap data with pgRouting, replacing BRouter. The graph
-- is loaded by the import-osm command.
create extension if not exists postgis;
create extension if not exists pgrouting;

-- A section of a foot accessible OpenStreetMap way between two junctions or
-- ends. Vertices are OpenStreetMap node IDs, so the graphs of neighboring
-- regions connect.
create table routing_edges (
  id bigint generated always as identity primary key,
  -- The imported region the edge belongs to.
  region text not null,
  osm_way_id bigint not null,
  source bigint not null,
  target bigint not null,
  -- The path from source to target.
  geom geometry(LineString, 4326) not null,
  -- The ground elevation at each point of geom, in meters.
  elevations real[] not null,
  length_meters double precision not null,
  -- The elevation gained and lost from source to target, in meters.
  climb_meters real not null,
  descent_meters real not null,
  -- OpenStreetMap tags that affect routing.
  highway text not null,
  surface text not null,
  sac_scale text not null,
  -- The terrain classified from the tags: paved, gravel, trail, technical or
  -- empty if unknown.
  terrain text not null
);

create index routing_edges_geom on routing_edges using gist (geom);
create index routing_edges_region on routing_edges (region);

-- The regions that have been imported.
create table routing_regions (
  name text primary key,
  -- The area the region covers. Edges may extend slightly beyond it.
  bounds geometry(Polygon, 4326) not null,
  edge_count integer not null,
  import_time timestamptz not null default current_timestamp
);
