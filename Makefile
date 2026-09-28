-include .env
export

OPENAPI_SPEC  := contracts/openapi/trip-service.openapi.yaml
GEN_DIR       := internal/generated
OPERATION_IDS := createTrip,getTrip,finishTrip,health,ready
GOOSE         := go tool goose -dir migrations

.PHONY: generate run build test migrate migrate-up migrate-down migrate-create

generate:
	mkdir -p $(GEN_DIR)
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-include-operation-ids $(OPERATION_IDS) \
		-o $(GEN_DIR)/api.gen.go \
		$(OPENAPI_SPEC)

run:
	go run ./cmd/trip-service

build:
	go build -o bin/trip-service ./cmd/trip-service

test:
	go test -race ./...

migrate: migrate-up

migrate-up:
	$(GOOSE) postgres "$(DATABASE_URL)" up

migrate-down:
	$(GOOSE) postgres "$(DATABASE_URL)" down

migrate-create:
	@test -n "$(name)" || (echo "usage: make migrate-create name=trips"; exit 1)
	$(GOOSE) -s create $(name) sql
