SWAG := $(shell go env GOPATH)/bin/swag

.PHONY: build run-api run-relay migrate swagger clean test load-test vet lint fmt fmt-check

## build: Build all binaries
build: swagger
	go build -o bin/api ./cmd/api
	go build -o bin/outbox-relay ./cmd/outbox-relay
	go build -o bin/migrate ./cmd/migrate

## migrate: Apply schema migrations (uses MIGRATION_DATABASE_URL, else DATABASE_URL)
migrate:
	go run ./cmd/migrate

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

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
