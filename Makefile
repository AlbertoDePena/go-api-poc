SWAG := $(shell go env GOPATH)/bin/swag

.PHONY: build run swagger clean test load-test aspire aspire-stop vet lint fmt fmt-check

## build: Build the API binary
build: swagger
	go build -o bin/api ./cmd/api

## run: Generate swagger docs and run the API server
run: swagger
	go run ./cmd/api

## swagger: Generate Swagger documentation
swagger:
	$(SWAG) init -g cmd/api/main.go -o docs

## vet: run go vet
vet:
	$(GO) vet ./...

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
