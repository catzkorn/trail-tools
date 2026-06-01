NPM_TAG := 25.1.0-alpine
DEX_TAG := v2.41.1-distroless
PG_TAG  := 18
CONTAINER_BIN ?= $(shell if command -v docker >/dev/null 2>&1; then printf '%s' docker; elif command -v container >/dev/null 2>&1; then printf '%s' container; fi)
CPUS ?= $(shell (nproc --all || sysctl -n hw.ncpu) 2>/dev/null || echo 1)
MAKEFLAGS += --jobs=$(CPUS)

# Run `make` with no target to print the help menu.
.DEFAULT_GOAL := help

ifeq ($(strip $(CONTAINER_BIN)),)
$(error neither docker nor container is available in PATH)
endif

# Command that prints the names of currently running containers, one per line.
# docker uses `ps`; Apple's `container` uses `ls` (whose ID column is the name).
ifeq ($(notdir $(CONTAINER_BIN)),docker)
CONTAINER_RUNNING := $(CONTAINER_BIN) ps --format '{{.Names}}'
else
CONTAINER_RUNNING := $(CONTAINER_BIN) ls --quiet
endif

UID_GID := $(shell id -u):$(shell id -g)

# Boilerplate for running a one-off node tool against ./web in a container, as
# the host user so written files aren't owned by root. Append the command, e.g.
# `$(NODE_RUN) npx eslint`.
NODE_RUN := $(CONTAINER_BIN) run --rm \
	-v $(CURDIR)/web:/srv \
	--user $(UID_GID) \
	-w /srv \
	-e NPM_CONFIG_CACHE=/srv/node_modules/.npm \
	-e NODE_OPTIONS='--disable-warning=ExperimentalWarning' \
	-e BROWSERSLIST_IGNORE_OLD_DATA=true \
	node:$(NPM_TAG)

# Flags shared by the `run` and `watch` backend invocations.
SERVER_FLAGS := \
	-database-url postgres://postgres:password@localhost:5432/postgres?sslmode=disable \
	-log-level debug \
	-oidc-client-id trail-tools-test \
	-oidc-client-secret abracadabra \
	-oidc-issuer-url http://0.0.0.0:5556/dex

# Production asset builds, shared by the standalone targets and by `gen`.
ESBUILD_PROD := go tool esbuild web/index.tsx --minify --bundle --outdir=web/dist --sourcemap --target=es6
TAILWIND_PROD := $(NODE_RUN) npx tailwindcss --minify -i base.css -o dist/index.css

# Block until dex is accepting TCP connections on its port.
# bash's /dev/tcp needs no extra tooling; fail after ~30s.
define WAIT_DEX
echo "waiting for dex on :5556..."; n=0; \
until bash -c 'exec 3<>/dev/tcp/localhost/5556' 2>/dev/null; do \
	n=$$((n+1)); [ $$n -ge 150 ] && { echo "dex not ready after 30s" >&2; exit 1; }; \
	sleep 0.2; \
done; echo "dex ready"
endef

# Block until postgres accepts TCP queries. pg_isready runs inside the container.
# -h localhost forces a TCP check: the entrypoint's bootstrap server listens only
# on the unix socket, so a default socket check would falsely report ready early.
define WAIT_DB
echo "waiting for postgres on :5432..."; n=0; \
until $(CONTAINER_BIN) exec postgres pg_isready -h localhost -U postgres >/dev/null 2>&1; do \
	n=$$((n+1)); [ $$n -ge 150 ] && { echo "postgres not ready after 30s" >&2; exit 1; }; \
	sleep 0.2; \
done; echo "postgres ready"
endef

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

.PHONY: db
db: ## Start the postgres container
	@if $(CONTAINER_RUNNING) | grep -qx postgres; then \
		echo "postgres already running"; \
	else \
		$(CONTAINER_BIN) run --rm --name postgres -p 5432:5432 -d -e POSTGRES_PASSWORD=password postgres:$(PG_TAG); \
	fi

.PHONY: sqlc
sqlc:
	go tool sqlc generate

.PHONY: buf
buf:
	go tool buf generate

.PHONY: buf-lint
buf-lint:
	cd api && go tool buf lint

.PHONY: buf-format
buf-format:
	cd api && go tool buf format -w

.PHONY: go-format
go-format:
	grep -L -R "^// Code generated .* DO NOT EDIT\.$$" --exclude-dir=.git --include="*.go" . | xargs go tool gofumpt -w

.PHONY: go-lint
go-lint:
	go tool honnef.co/go/tools/cmd/staticcheck ./...

.PHONY: dex
dex: ## Start the dex OIDC container
	@if $(CONTAINER_RUNNING) | grep -qx dex; then \
		echo "dex already running"; \
	else \
		$(CONTAINER_BIN) run \
			--rm \
			--name dex \
			-d \
			-v $(CURDIR)/test/dex/dex-config.yaml:/etc/dex/config.docker.yaml \
			-p 5556:5556 dexidp/dex:$(DEX_TAG); \
	fi

.PHONY: run
run: dex db ## Run the backend (starts db + dex)
	@$(WAIT_DEX)
	@$(WAIT_DB)
	go run main.go $(SERVER_FLAGS)

.PHONY: web-deps
web-deps: ## Install web dependencies
	$(NODE_RUN) npm install

.PHONY: tsc
tsc:
	$(NODE_RUN) npx tsc --noEmit

.PHONY: eslint
eslint:
	$(NODE_RUN) npx eslint

.PHONY: web-lint
web-lint: tsc eslint

.PHONY: web-format
web-format:
	$(NODE_RUN) npx prettier --write .

.PHONY: esbuild
esbuild:
	$(ESBUILD_PROD)

.PHONY: tailwindcss
tailwindcss:
	$(TAILWIND_PROD)

.PHONY: watch
watch: dex db ## Run backend + assets with live reload
	@$(WAIT_DEX)
	@$(WAIT_DB)
	-/usr/bin/env bash -c "\
		trap 'kill %1 %2' EXIT;\
		go tool esbuild web/index.tsx \
			--bundle \
			--outdir=web/dist \
			--sourcemap \
			--target=es6 \
			--watch=forever 2> /dev/null & \
		$(NODE_RUN) npx -s tailwindcss -i base.css -o dist/index.css --watch & \
		go tool github.com/mitranim/gow -s run main.go $(SERVER_FLAGS) -serve-dir $(CURDIR)/web/dist \
	"

.PHONY: gen
# `buf` has `clean: true`, so `buf generate` briefly deletes the gen/ files that
# esbuild bundles. Keep them off the same parallel batch: depend on the codegen
# (which can run in parallel), then build the assets in the recipe, which only
# runs once all prerequisites have finished.
gen: sqlc buf ## Run all code generation
	$(ESBUILD_PROD)
	$(TAILWIND_PROD)

.PHONY: lint
lint: web-lint buf-lint go-lint ## Run all linters

.PHONY: format
format: go-format web-format buf-format ## Run all formatters

.PHONY: clean
clean: ## Stop the db and dex containers
	@for c in postgres dex; do \
		if $(CONTAINER_RUNNING) | grep -qx $$c; then \
			echo "stopping $$c"; $(CONTAINER_BIN) stop $$c >/dev/null; \
		else \
			echo "$$c not running"; \
		fi; \
	done
