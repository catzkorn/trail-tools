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
- **Routes**: draw routes by placing waypoints that snap to trails
  ([BRouter](https://brouter.de)), import GPX files or turn a past activity into
  a route. Each route has an elevation profile, distance, gain, grades and a
  terrain breakdown, and can be downloaded as GPX for your watch. Overlays show
  your personal heatmap, OpenStreetMap trails and points of interest.
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
then the detailed streams of each activity, most recent first. Strava's API
rate limits (100 requests per 15 minutes, 1000 per day) mean a long history
takes a few days to sync fully; the sync pauses and resumes automatically, and
the app syncs new activities every few hours.

### AI agents (MCP)

The MCP server is at `http://localhost:8080/mcp` (streamable HTTP). Create an
API token under Settings, then for example in Claude Code:

```sh
$ claude mcp add --transport http trail-tools http://localhost:8080/mcp --header "Authorization: Bearer <token>"
```

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

### Map data services

Routing, trails and elevation come from public services that are fine for
personal use. Their URLs can be changed with flags, e.g. to self host BRouter:

| Flag             | Default                         | Used for                        |
| ---------------- | ------------------------------- | ------------------------------- |
| `-brouter-url`   | `https://brouter.de/brouter`    | Snapping routes to trails       |
| `-overpass-urls` | Public Overpass API instances   | Trails and points of interest   |

Public Overpass instances are often overloaded; trail and point of interest
lookups return an error asking to retry when they all fail.

## Deploying

Trail Tools runs on [Cloud Run](https://cloud.google.com/run) with a
[Neon](https://neon.com) serverless Postgres database, sized to stay within
the free tiers for personal use: Cloud Run scales to zero, runs at most one
instance and only uses CPU during requests, and Neon suspends the database
when it's idle.

The Google Cloud configuration is managed with Terraform in
[`infra/`](infra), with its state in a Cloud Storage bucket. To apply changes
locally:

```sh
$ export GOOGLE_OAUTH_ACCESS_TOKEN=$(gcloud auth print-access-token)
$ terraform -chdir=infra init
$ terraform -chdir=infra apply
```

### Secrets

Secrets are created by Terraform, but their values are added by hand so they
aren't stored in the Terraform state:

- `database-url`: the direct (not pooled) Neon connection string, as
  migrations need session advisory locks.

  ```sh
  $ pbpaste | gcloud secrets versions add database-url --data-file=- --project trail-tools-444904
  ```

- `admin-bootstrap-token`: works as an invitation to create the first
  administrator at `/invite/<token>`, until there is one.

  ```sh
  $ openssl rand -hex 24 | gcloud secrets versions add admin-bootstrap-token --data-file=- --project trail-tools-444904
  $ gcloud secrets versions access latest --secret admin-bootstrap-token --project trail-tools-444904
  ```

  Passkeys are bound to the domain, so create the administrator on the
  domain you'll use.

- `strava-client-secret`: the Strava API application's client secret. Also
  set `strava_client_id` in [`terraform.tfvars`](infra/terraform.tfvars),
  which enables Strava and an hourly Cloud Scheduler job that syncs
  activities, and set the Strava app's callback domain.

### Custom domain

The app is served on `trailrunn.ing`, set as `domain` in
[`terraform.tfvars`](infra/terraform.tfvars). To set up a domain:

1. Verify the domain with `gcloud domains verify <domain>`.
2. Set `domain_enabled = true` in [`terraform.tfvars`](infra/terraform.tfvars)
   and apply it locally with `terraform -chdir=infra apply`. Only a verified
   owner of the domain can map it, so the first apply runs with your own
   credentials.
3. Create the DNS records from `terraform -chdir=infra output domain_records`.
   Cloud Run provisions and renews the TLS certificate, which takes up to
   about 15 minutes after the records resolve.

### Bootstrapping a new project

The state bucket and the billing permission are created by hand once:

```sh
$ gcloud storage buckets create gs://<project>-tfstate --location us-west1 --uniform-bucket-level-access --public-access-prevention
$ gcloud storage buckets update gs://<project>-tfstate --versioning
```

Then update the bucket in [`versions.tf`](infra/versions.tf) and the
variables in [`terraform.tfvars`](infra/terraform.tfvars), and apply locally.
