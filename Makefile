NPM_TAG:=23.7.0-alpine
DEX_TAG:=v2.41.1-distroless
CPUS ?= $(shell (nproc --all || sysctl -n hw.ncpu) 2>/dev/null || echo 1)
MAKEFLAGS += --jobs=$(CPUS)

.PHONY: db
db:
	-docker run --rm -it --name postgres -p 5432:5432 -d -e POSTGRES_PASSWORD=password postgres:latest && sleep 5

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

.PHONY: dex
dex:
	-docker run \
		--rm \
		--name dex \
		-d \
		-v $(shell pwd)/test/dex/dex-config.yaml:/etc/dex/config.docker.yaml \
		-p 5556:5556 dexidp/dex:v2.41.1-distroless

.PHONY: run
run: dex db
	go run main.go \
		-database-url postgres://postgres:password@localhost:5432/postgres?sslmode=disable\
		-log-level debug \
		-oidc-client-id trail-tools-test \
		-oidc-client-secret abracadabra \
		-oidc-issuer-url http://0.0.0.0:5556/dex

.PHONY: web-deps
web-deps:
	docker run \
		--rm \
		-v $(shell pwd)/web:/srv \
		--user $(shell id -u):$(shell id -g) \
		-w /srv \
		-e NPM_CONFIG_CACHE=/srv/node_modules/.npm \
		-e NODE_OPTIONS='--disable-warning=ExperimentalWarning' \
		node:$(NPM_TAG) npm install

.PHONY: tsc
tsc:
	docker run \
		--rm \
		-v $(shell pwd)/web:/srv \
		--user $(shell id -u):$(shell id -g) \
		-w /srv -e NPM_CONFIG_CACHE=/srv/node_modules/.npm \
		-e NODE_OPTIONS='--disable-warning=ExperimentalWarning' \
		node:$(NPM_TAG) npx tsc --noEmit

.PHONY: eslint
eslint:
	docker run \
		--rm \
		-v $(shell pwd)/web:/srv \
		--user $(shell id -u):$(shell id -g) \
		-w /srv \
		-e NPM_CONFIG_CACHE=/srv/node_modules/.npm \
		-e NODE_OPTIONS='--disable-warning=ExperimentalWarning' \
		node:$(NPM_TAG) npx eslint

.PHONY: web-lint
web-lint: tsc eslint

.PHONY: web-format
web-format:
	docker run \
		--rm \
		-v $(shell pwd)/web:/srv \
		--user $(shell id -u):$(shell id -g) \
		-w /srv -e NPM_CONFIG_CACHE=/srv/node_modules/.npm \
		-e NODE_OPTIONS='--disable-warning=ExperimentalWarning' \
		node:$(NPM_TAG) npx prettier --write .

.PHONY: esbuild
esbuild:
	go tool esbuild web/index.tsx \
		--minify \
		--bundle \
		--outdir=web/dist \
		--sourcemap \
		--target=es6

.PHONY: tailwindcss
tailwindcss:
	docker run \
		--rm \
		-v $(shell pwd)/web:/srv \
		--user $(shell id -u):$(shell id -g) \
		-w /srv \
		-e NPM_CONFIG_CACHE=/srv/node_modules/.npm \
		-e NODE_OPTIONS='--disable-warning=ExperimentalWarning' \
		node:$(NPM_TAG) npx tailwindcss --minify -i base.css -o dist/index.css

.PHONY: watch
watch: dex db
	-/usr/bin/env bash -c "\
		trap 'kill %1 %2' EXIT;\
		go tool esbuild web/index.tsx \
			--bundle \
			--outdir=web/dist \
			--sourcemap \
			--target=es6 \
			--watch=forever & \
		docker run \
			-t \
			--rm \
			-v $(shell pwd)/web:/srv \
			--user $(shell id -u):$(shell id -g) \
			-w /srv \
			-e NPM_CONFIG_CACHE=/srv/node_modules/.npm \
			-e NODE_OPTIONS='--disable-warning=ExperimentalWarning' \
			node:$(NPM_TAG) npx -s tailwindcss -i base.css -o dist/index.css --watch & \
		go run main.go \
			-database-url postgres://postgres:password@localhost:5432/postgres?sslmode=disable \
			-log-level debug \
			-oidc-client-id trail-tools-test \
			-oidc-client-secret abracadabra \
			-oidc-issuer-url http://0.0.0.0:5556/dex \
			-serve-dir $(shell pwd)/web/dist \
	"

.PHONY: gen
gen: sqlc buf esbuild tailwindcss

.PHONY: lint
lint: web-lint buf-lint

.PHONY: format
format: go-format web-format buf-format
