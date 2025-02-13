NPM_TAG:=23.7.0-alpine
DEX_TAG:=v2.41.1-distroless

.PHONY: db-up
db-up:
	docker run --rm -it --name postgres -p 5432:5432 -d -e POSTGRES_PASSWORD=password postgres:latest

.PHONY: db-down
db-down:
	docker stop postgres

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

.PHONY: run
run:
	-go run main.go \
		-database-url postgres://postgres:password@localhost:5432/postgres?sslmode=disable\
		-log-level debug \
		-oidc-client-id $$OIDC_CLIENT_ID \
		-oidc-client-secret $$OIDC_CLIENT_SECRET \
		-oidc-issuer-url $$OIDC_ISSUER_URL

.PHONY: dex
dex:
	docker run \
		--rm \
		-v $$(pwd)/test/dex/dex-config.yaml:/etc/dex/config.docker.yaml \
		-p 5556:5556 dexidp/dex:v2.41.1-distroless

.PHONY: web-deps
web-deps:
	docker run \
		--rm \
		-v $$(pwd)/web:/srv \
		--user $$(id -u):$$(id -g) \
		-w /srv \
		-e NPM_CONFIG_CACHE=/srv/node_modules/.npm \
		-e NODE_OPTIONS='--disable-warning=ExperimentalWarning' \
		node:$(NPM_TAG) npm install

.PHONY: web-lint
web-lint:
	docker run \
		--rm \
		-v $$(pwd)/web:/srv \
		--user $$(id -u):$$(id -g) \
		-w /srv -e NPM_CONFIG_CACHE=/srv/node_modules/.npm \
		-e NODE_OPTIONS='--disable-warning=ExperimentalWarning' \
		node:$(NPM_TAG) npx tsc --noEmit
	docker run \
		--rm \
		-v $$(pwd)/web:/srv \
		--user $$(id -u):$$(id -g) \
		-w /srv \
		-e NPM_CONFIG_CACHE=/srv/node_modules/.npm \
		-e NODE_OPTIONS='--disable-warning=ExperimentalWarning' \
		node:$(NPM_TAG) npx eslint

.PHONY: web-format
web-format:
	docker run \
		--rm \
		-v $$(pwd)/web:/srv \
		--user $$(id -u):$$(id -g) \
		-w /srv -e NPM_CONFIG_CACHE=/srv/node_modules/.npm \
		-e NODE_OPTIONS='--disable-warning=ExperimentalWarning' \
		node:$(NPM_TAG) npx prettier --write .

.PHONY: web
web:
	go tool esbuild web/index.tsx \
		--minify \
		--bundle \
		--outdir=web/dist \
		--sourcemap \
		--target=es6
	docker run \
		--rm \
		-v $$(pwd)/web:/srv \
		--user $$(id -u):$$(id -g) \
		-w /srv \
		-e NPM_CONFIG_CACHE=/srv/node_modules/.npm \
		-e NODE_OPTIONS='--disable-warning=ExperimentalWarning' \
		node:$(NPM_TAG) npx tailwindcss --minify -i base.css -o dist/index.css

.PHONY: watch
watch:
	-/usr/bin/env bash -c "\
		trap 'kill %1 %2' EXIT;\
		go tool esbuild web/index.tsx \
			--bundle \
			--outdir=web/dist \
			--sourcemap \
			--target=es6 \
			--minify \
			--watch=forever & \
		docker run \
			-t \
			--rm \
			-v $$(pwd)/web:/srv \
			--user $$(id -u):$$(id -g) \
			-w /srv \
			-e NPM_CONFIG_CACHE=/srv/node_modules/.npm \
			-e NODE_OPTIONS='--disable-warning=ExperimentalWarning' \
			node:$(NPM_TAG) npx -s tailwindcss --minify -i base.css -o dist/index.css --watch & \
		go run main.go \
			-database-url postgres://postgres:password@localhost:5432/postgres?sslmode=disable \
			-log-level debug \
			-oidc-client-id $$OIDC_CLIENT_ID \
			-oidc-client-secret $$OIDC_CLIENT_SECRET \
			-oidc-issuer-url $$OIDC_ISSUER_URL \
			-serve-dir $$(pwd)/web/dist \
	"

.PHONY: gen
gen: sqlc buf web

.PHONY: lint
lint: web-lint buf-lint

.PHONY: format
format: go-format web-format buf-format
