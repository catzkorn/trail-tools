SQLC_VERSION:=1.27.0
BUF_VERSION:=1.47.2

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
	docker run --rm -v $$(pwd):/srv --user $(id -u):$(id -g) -w /srv bufbuild/buf:$(BUF_VERSION) generate

.PHONY: gen
gen: sqlc buf

.PHONY: run
run:
	go run main.go -database-url postgres://postgres:password@localhost:5432/postgres?sslmode=disable
