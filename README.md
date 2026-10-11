# Trail tools

A trail running planner: sync your activities from Strava, plan your training
for up to a year towards your goal races, design routes on a topo map, check
the weather along a route and build an opinionated race day plan you can
share with your crew.

Everything is also exposed as an [MCP](https://modelcontextprotocol.io) server,
so you can bring your own AI agent to do the planning.

## Features

- **Training plans**: plans of up to a year with goal races, planned at the
  month (phase and volume), week (focus and targets) and day (workouts) level,
  compared with what you actually ran.
- **Routes**: a Strava style route editor where waypoints snap to trails or
  roads (routed with [pgRouting](https://pgrouting.org) on
  [OpenStreetMap](https://www.openstreetmap.org) data), prefer the paths you've run before
  (your heatmap), or are joined by straight lines for off-trail sections, mixed
  within one route. Drag the route to reshape it, undo and redo, close the loop
  or turn it into an out and back. You can also import GPX files or turn a past
  activity into a route. Each route has an elevation profile, distance, gain,
  grades and a terrain breakdown, and can be downloaded as GPX for your watch.
  Overlays show your personal heatmap, OpenStreetMap trails and points of
  interest.
- **Weather**: an hourly forecast for where you'll be along a route, given a
  start time and pace, with warnings for thunderstorms, freezing levels, wind
  and darkness ([Open-Meteo](https://open-meteo.com)). Works for saved routes
  and uploaded GPX files.
- **Race plans**: goals, checkpoints with cutoffs, pacing, fueling, gear, crew,
  contingencies, race morning and mental strategy. The plan is checked against
  a set of rules (e.g. cutoff buffers, carbohydrate per segment, a headlamp if
  you'll be out in the dark) and can be shared read-only with a link.
- **Activities**: your full Strava history including GPS, elevation and heart
  rate streams, stored locally.

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
then the details, photos and detailed streams of each activity, most recent
first. Photos stay on Strava: only their captions, locations and links are
stored, and they're only shown to you. Strava's API rate limits (100 requests
per 15 minutes, 1000 per day) mean a long history takes a few days to sync
fully; the sync pauses and resumes automatically, and the app syncs new
activities every few hours. The activities page checks for new activities
every minute while it's open.

When the server is reachable from the internet, Strava can also push new,
changed and deleted activities to it as they happen, so they appear within
seconds: set `-strava-webhook-verify-token` to a random secret, and the
server subscribes to Strava's webhook events at `/strava/webhook` when it
starts. Strava allows one subscription per API application, so this replaces
a subscription to another server. Strava can't reach a local server, so leave
it unset locally: new activities arrive with the periodic sync, or when you
sync from Settings.

### AI agents (MCP)

The MCP server is at `<public URL>/mcp`, e.g. `http://localhost:8080/mcp`
(streamable HTTP). Clients connect with OAuth or an API token.

#### claude.ai, Claude Desktop and other OAuth clients

Custom connectors in claude.ai and Claude Desktop, ChatGPT and other remote MCP
clients sign in with OAuth:

1. In claude.ai, open Settings → Connectors → Add custom connector, and enter
   `https://<your domain>/mcp` as the URL. Leave the OAuth client ID and
   secret empty: clients register themselves.
2. Connect. You're sent to Trail Tools, where you sign in with your passkey
   and approve the connection.
3. The connector is now available in claude.ai, and in the Claude apps signed
   in to the same account.

Connected apps are listed under Settings, where you can disconnect them. Their
access tokens last 30 days and are refreshed automatically; an app that isn't
used for 90 days needs to be connected again.

claude.ai connects from Anthropic's servers, so the server must be reachable
from the internet over HTTPS at the URL you enter, as it's the OAuth issuer
and resource. The server's URL is `https://<hostname>:<port>`, from
`-hostname`, `-port` and the TLS flags.

How it works: `/mcp` answers unauthenticated requests with a
`WWW-Authenticate` header pointing at the protected resource metadata
(`/.well-known/oauth-protected-resource/mcp`), which points at the
authorization server metadata (`/.well-known/oauth-authorization-server`).
Clients register dynamically at `/oauth/register`, without anything being
stored: the registration is encoded in the client ID. Only redirect URIs of
known clients are accepted: loopback addresses, HTTPS on an allowlist
(claude.ai, claude.com, chatgpt.com, vscode.dev) and a few desktop app schemes
(see `internal/oauth`). `/oauth/authorize` requires PKCE (S256) and shows a
consent page, and `/oauth/token` issues opaque access tokens and rotating
refresh tokens, stored hashed.

#### API tokens

Clients that support custom headers can use an API token instead. Create one
under Settings, then for example in Claude Code:

```sh
$ claude mcp add --transport http trail-tools http://localhost:8080/mcp --header "Authorization: Bearer <token>"
```

Without the header, Claude Code connects with OAuth instead.

#### Tools

The MCP tools are generated from the Connect API at startup: every RPC is a
tool with the same name, its input schema comes from the request message and
its description from the proto comments. Methods annotated with
`option (mcp.v1.method).exclude = true` (e.g. API token management) are not
exposed. Service comments are sent to the agent as the server's instructions,
so the proto comments are the agent's documentation; write them for an agent.

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

### Adding an RPC

1. Add the RPC to the service in `api/<domain>/v1/*.proto`, with comments
   describing it for both humans and agents.
2. Run `make gen`. This generates the Go and TypeScript code and the embedded
   descriptor set used for the MCP tool descriptions.
3. Implement the method in `internal/services/<domain>`. It's automatically
   exposed over MCP.

### Routing data

Routes are routed in Postgres with [PostGIS](https://postgis.net) and
[pgRouting](https://pgrouting.org), on the foot accessible ways of
OpenStreetMap data imported into the database. `make db` builds a Postgres
image with both extensions.

Administrators manage the imported areas on the map data page, at
`/map-data` or under the user menu. It shows each area with the space it
takes and the size of the database, deletes areas to free space, and imports
the area shown in the middle of the map. The server picks the
[Geofabrik](https://download.geofabrik.de) extracts to read from their
outlines in Geofabrik's index: at points across the area, the smallest
extract containing each, so an area across a border reads the extracts on
both sides, up to 4. It reads only the paths inside the area, and looks up
elevations from the SRTM derived tiles of the
[AWS Terrain Tiles](https://registry.opendata.aws/terrain-tiles/) dataset.
Imports run in the server.

Imports can also be run from the command line, e.g. to import a whole
extract:

```sh
$ make import-osm REGION=liechtenstein PBF=https://download.geofabrik.de/europe/liechtenstein-latest.osm.pbf
```

Use `BOUNDS=south,west,north,east` to only import part of an extract, and
separate the extracts of neighboring countries with commas in `PBF` to import
an area across a border. Importing a region again replaces it.

Storage grows with the area and density of paths: all of Switzerland takes
about 715 MB, a 100 by 100 km box of it about 230 MB and Liechtenstein about
4.5 MB. Neon's free plan has 0.5 GB of storage. Memory grows with the area
imported, as only the paths inside it are kept: a 100 km² area of the
Switzerland extract peaks at under 400 MB.

### Map data services

Trails, points of interest, place search, elevations of straight legs and
weather come from public services that are fine for personal use:

| Flag             | Default                         | Used for                        |
| ---------------- | ------------------------------- | ------------------------------- |
| `-overpass-urls` | Public Overpass API instances   | Trails and points of interest   |

Public Overpass instances are often overloaded; trail and point of interest
lookups return an error asking to retry when they all fail.
