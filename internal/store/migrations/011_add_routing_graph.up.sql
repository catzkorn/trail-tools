-- Routing on OpenStreetMap data with pgRouting. The graph is loaded by the
-- import-osm command. Columns are kept small, as a graph
-- has millions of edges.
create extension if not exists postgis;
create extension if not exists pgrouting;

-- The regions that have been imported.
create table routing_regions (
  id smallint generated always as identity primary key,
  name text unique not null,
  -- The area the region covers. Edges may extend slightly beyond it.
  bounds geometry(Polygon, 4326) not null,
  edge_count integer not null,
  import_time timestamptz not null default current_timestamp
);

-- A section of a foot accessible OpenStreetMap way between two junctions or
-- ends. Vertices are OpenStreetMap node IDs, so the graphs of neighboring
-- regions connect.
create table routing_edges (
  id bigint generated always as identity primary key,
  region_id smallint not null references routing_regions (id) on delete cascade,
  source bigint not null,
  target bigint not null,
  -- The path from source to target.
  geom geometry(LineString, 4326) not null,
  -- The ground elevation at each point of geom, in meters.
  elevations smallint[] not null,
  length_meters real not null,
  -- The elevation gained and lost from source to target, in meters.
  climb_meters real not null,
  descent_meters real not null,
  -- OpenStreetMap tags that affect routing.
  highway text not null,
  sac_scale text not null,
  -- The terrain classified from the tags: paved, gravel, trail, technical or
  -- empty if unknown.
  terrain text not null
);

create index routing_edges_geom on routing_edges using gist (geom);
create index routing_edges_region on routing_edges (region_id);
