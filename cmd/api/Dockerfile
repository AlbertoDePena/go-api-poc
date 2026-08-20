# --- swagger stage ---
FROM golang:1.23 AS swagger
RUN go install github.com/swaggo/swag/cmd/swag@latest
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN swag init -g cmd/api/main.go -o api/docs

# --- go build stage ---
FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=swagger /src/api/docs ./api/docs
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/api ./cmd/api

# --- final stage ---
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/api /api
EXPOSE 8080
ENTRYPOINT ["/api"]
