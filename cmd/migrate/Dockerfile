# --- go build stage ---
FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# --- final stage ---
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/migrate /migrate
ENTRYPOINT ["/migrate"]
