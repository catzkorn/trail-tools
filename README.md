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
