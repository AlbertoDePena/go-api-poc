# --- go build stage ---
FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/outbox-relay ./cmd/outbox-relay

# --- final stage ---
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/outbox-relay /outbox-relay
ENTRYPOINT ["/outbox-relay"]
