# Postgres with PostGIS and pgRouting for routing on OpenStreetMap data, for
# local development and tests.
ARG PG_TAG=18
FROM postgres:${PG_TAG}
ARG PG_MAJOR=18
RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
		postgresql-${PG_MAJOR}-postgis-3 \
		postgresql-${PG_MAJOR}-pgrouting \
	&& rm -rf /var/lib/apt/lists/*
# Install the extensions in the template database, so that databases created
# for tests have them without each installing them again.
COPY template-extensions.sql /docker-entrypoint-initdb.d/
