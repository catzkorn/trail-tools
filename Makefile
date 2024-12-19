SQLC_VERSION:=1.27.0
BUF_VERSION:=1.47.2
NPM_TAG:=23-alpine
ESBUILD_VERSION:=0.24.0
POSTGRES_VERSION:=17

.PHONY: db-up
db-up:
	docker run --rm -it --name postgres -p 5432:5432 -d -e POSTGRES_PASSWORD=password postgres:latest

.PHONY: db-down
db-down:
	docker stop postgres

.PHONY: sqlc
sqlc:
	docker run --rm -v $$(pwd):/srv --user $(id -u):$(id -g) -w /srv sqlc/sqlc:$(SQLC_VERSION) generate

.PHONY: buf
buf:
	go run github.com/bufbuild/buf/cmd/buf@v$(BUF_VERSION) generate

.PHONY: gen
gen: sqlc buf web

.PHONY: run
run:
	go run main.go -database-url postgres://postgres:password@localhost:5432/postgres?sslmode=disable -oidc-client-id $$OIDC_CLIENT_ID -oidc-client-secret $$OIDC_CLIENT_SECRET

.PHONY: web-deps
web-deps:
	docker run --rm -v $$(pwd)/web:/srv --user $$(id -u):$$(id -g) -w /srv -e NODE_OPTIONS='--disable-warning=ExperimentalWarning' node:$(NPM_TAG) npm install

.PHONY: web
web:
	go run github.com/evanw/esbuild/cmd/esbuild@v0.24.0 web/index.tsx --minify --bundle --outdir=web/dist --sourcemap --target=es6
	docker run --rm -v $$(pwd)/web:/srv --user $$(id -u):$$(id -g) -w /srv d3fk/tailwindcss:latest --minify -i base.css -o dist/index.css
