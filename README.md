# Trail tools

A website that is both an example of a full stack Go/React/TS/Postgres app and a bunch of useful trail running tools!

## Prerequisites

- Go
- Docker or [Apple `container`](https://github.com/apple/container)

## Running

To start Postgres and run the server locally, use:

```sh
$ make run
```

Then open http://localhost:8080/invite/local-admin to create the
administrator account with a passkey. Trail Tools is invite only: invite other
users, or yourself to add a passkey on another device, from the settings page.

### Strava

To sync activities, create a Strava API application at
https://www.strava.com/settings/api with the "Authorization Callback Domain"
set to `localhost`, then run the server with its credentials:

```sh
$ STRAVA_CLIENT_ID=12345 STRAVA_CLIENT_SECRET=... make run
```

Connect your account under Settings. The first sync fetches all activities,
then the detailed streams of each activity, most recent first. Strava's API
rate limits (100 requests per 15 minutes, 1000 per day) mean a long history
takes a few days to sync fully; the sync pauses and resumes automatically, and
the app syncs new activities every few hours.

## Development

To run the server with auto-rebuild of the backend, frontend and CSS, run:

```sh
$ make watch
```

To run the tests (starts Postgres; each test gets its own database):

```sh
$ make test
```

To view available make targets, run:

```sh
$ make help
```

### Routing data

Routes are routed in Postgres with [PostGIS](https://postgis.net) and
[pgRouting](https://pgrouting.org), on the foot accessible ways of
OpenStreetMap data imported into the database. `make db` builds a Postgres
image with both extensions.

Administrators manage the imported areas on the map data page, at
`/map-data` or under the user menu. It shows each area with the space it
takes and the size of the database, deletes areas to free space, and imports
the area shown in the middle of the map. The server picks the smallest
[Geofabrik](https://download.geofabrik.de) extract covering the area, reads
only the paths inside it, and looks up elevations from the SRTM derived tiles
of the [AWS Terrain Tiles](https://registry.opendata.aws/terrain-tiles/)
dataset. Imports run in the server.

Imports can also be run from the command line, e.g. to import a whole
extract:

```sh
$ make import-osm REGION=liechtenstein PBF=https://download.geofabrik.de/europe/liechtenstein-latest.osm.pbf
```

Use `BOUNDS=south,west,north,east` to only import part of an extract.
Importing a region again replaces it.

Storage grows with the area and density of paths: all of Switzerland takes
about 715 MB, a 100 by 100 km box of it about 230 MB and Liechtenstein about
4.5 MB. Neon's free plan has 0.5 GB of storage. Memory grows with the area
imported, as only the paths inside it are kept: a 100 km² area of the
Switzerland extract peaks at under 400 MB.
