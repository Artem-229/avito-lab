-include .env
export

OPENAPI_SPEC  := contracts/openapi/trip-service.openapi.yaml
GEN_DIR       := internal/generated
OPERATION_IDS := createTrip,getTrip,finishTrip,health,ready
GOOSE         := go tool goose -dir migrations
IMAGE         := trip-service:local
DOCKER_DB_URL := $(subst 127.0.0.1,host.docker.internal,$(subst localhost,host.docker.internal,$(DATABASE_URL)))

.PHONY: generate run build test test-integration migrate migrate-up migrate-down migrate-create docker-build docker-run lint

generate:
	mkdir -p $(GEN_DIR)
	go tool oapi-codegen \
		-generate types,chi-server,spec \
		-package api \
		-include-operation-ids $(OPERATION_IDS) \
		-o $(GEN_DIR)/api.gen.go \
		$(OPENAPI_SPEC)
	go generate ./internal/...

lint:
	golangci-lint run

run:
	go run ./cmd/service

build:
	go build -o bin/trip-service ./cmd/service

test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

test-integration:
	@test -n "$(TEST_DATABASE_URL)" || (echo "TEST_DATABASE_URL is required"; exit 1)
	$(GOOSE) postgres "$(TEST_DATABASE_URL)" up
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -tags=integration -race ./internal/repository/... ./internal/infra/postgres/transaction/...

migrate: migrate-up

migrate-up:
	$(GOOSE) postgres "$(DATABASE_URL)" up

migrate-down:
	$(GOOSE) postgres "$(DATABASE_URL)" down

migrate-create:
	@test -n "$(name)" || (echo "usage: make migrate-create name=trips"; exit 1)
	$(GOOSE) -s create $(name) sql

docker-build:
	docker build -f deploy/Dockerfile -t $(IMAGE) .

docker-run:
	docker run --rm --env-file .env \
		-e DATABASE_URL="$(DOCKER_DB_URL)" \
		-e HTTP_ADDR=:8080 \
		--add-host=host.docker.internal:host-gateway \
		-p 8080:8080 \
		$(IMAGE)
