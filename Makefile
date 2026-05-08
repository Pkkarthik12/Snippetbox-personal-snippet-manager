APP=snippetbox

.PHONY: run build test lint db-migrate db-reset

run:
	go run ./cmd/server

build:
	go build -o ./bin/$(APP) ./cmd/server

test:
	go test ./...

lint:
	golangci-lint run

db-migrate:
	psql "$$DATABASE_URL" -f schema/001_init.sql

db-reset:
	dropdb --if-exists snippetbox
	createdb snippetbox
	psql snippetbox -f schema/001_init.sql
