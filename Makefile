SWAG := $(shell go env GOPATH)/bin/swag

.PHONY: build run-api run-relay swagger clean test load-test aspire aspire-stop postgres postgres-stop vet lint fmt fmt-check docker-build docker-up docker-down

## build: Build all binaries
build: swagger
	go build -o bin/api ./cmd/api
	go build -o bin/outbox-relay ./cmd/outbox-relay

## run-api: Generate swagger docs and run the API server
run-api: swagger
	go run ./cmd/api

## run-relay: Run the outbox relay (separate binary; run alongside the API)
run-relay:
	go run ./cmd/outbox-relay

## swagger: Generate Swagger documentation
swagger:
	$(SWAG) init -g cmd/api/main.go -o api/docs

## vet: run go vet
vet:
	go vet ./...

## lint: go vet + gofmt check
lint: vet fmt-check

## fmt: format all Go source files
fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*')

## fmt-check: fail if any Go source file is not formatted
fmt-check:
	@out="$$(gofmt -l . | grep -v '^vendor/' | head -n 20)"; \
	if [ -n "$$out" ]; then \
		echo "The following files are not formatted:"; echo "$$out"; exit 1; \
	fi

## test: Run all tests
test:
	go test ./...

## load-test: run load test using k6
load-test:
	@echo "Running load test..."
	cat .load-test/script.js | docker run --rm -i grafana/k6 run -

## clean: Remove build artifacts
clean:
	rm -rf bin/ api/docs/

## deps: Install development dependencies
deps:
	go install github.com/swaggo/swag/cmd/swag@latest

## docker-build: Build Docker images
docker-build:
	docker compose build

## docker-up: Start all services via docker-compose
docker-up:
	docker compose up -d

## docker-down: Stop all services
docker-down:
	docker compose down

## aspire: Run .NET Aspire Dashboard for local OTel collection
aspire:
	docker run --rm -d \
		--name aspire-dashboard \
		-p 18888:18888 \
		-p 18889:18889 \
		-e DOTNET_DASHBOARD_UNSECURED_ALLOW_ANONYMOUS=true \
		mcr.microsoft.com/dotnet/aspire-dashboard:9.2

## aspire-stop: Stop the Aspire Dashboard
aspire-stop:
	docker stop aspire-dashboard

## postgres: Run a local PostgreSQL for `make run-api` / `make run-relay`
postgres:
	docker run --rm -d \
		--name api-poc-postgres \
		-p 5432:5432 \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_DB=apipoc \
		postgres:17-alpine

## postgres-stop: Stop the local PostgreSQL container
postgres-stop:
	docker stop api-poc-postgres

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
