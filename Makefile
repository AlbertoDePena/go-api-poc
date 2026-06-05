SWAG := $(shell go env GOPATH)/bin/swag

.PHONY: build run swagger clean test aspire aspire-stop

## build: Build the API binary
build: swagger
	go build -o bin/api ./cmd/api

## run: Generate swagger docs and run the API server
run: swagger
	go run ./cmd/api

## swagger: Generate Swagger documentation
swagger:
	$(SWAG) init -g cmd/api/main.go -o docs

## test: Run all tests
test:
	go test ./...

## clean: Remove build artifacts
clean:
	rm -rf bin/ docs/

## deps: Install development dependencies
deps:
	go install github.com/swaggo/swag/cmd/swag@latest

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

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
