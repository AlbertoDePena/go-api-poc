# Go API POC

A proof-of-concept Go service built as a **multi-binary repository** sharing a single
`internal/` library. It follows an opinionated, container-oriented layout: each
independently-deployable, independently-scalable responsibility is its own binary under
`cmd/`, while all business logic lives once in `internal/` and is reused across them.

Current binaries:

| Binary | Responsibility | Scaling profile |
| --- | --- | --- |
| `cmd/api` | HTTP/JSON API (chi + Swagger), OIDC-protected | N replicas behind a load balancer |
| `cmd/outbox-relay` | Polls the outbox table and dispatches pending events | Single active instance |

## Architecture

Dependencies point in one direction only:

```
handler (cmd/api)  ─▶  service  ─▶  repository / Transactor (interfaces)  ─▶  sqlite (impl)
                                └─▶  metrics / feature (interfaces + impls)
```

Nothing above the service layer knows a database exists. Handlers depend on
`service.GreetingService`; the service depends on `repository` interfaces and a
`repository.Transactor`; the SQLite adapter is the only package that imports a driver.

```
cmd/
  api/                     # HTTP API binary
    main.go                # wiring: config → otel → db → service → router
    router.go              # route mounting for this binary
    handler/               # HTTP handlers + edge types (Request/Response DTOs)
  outbox-relay/            # outbox relay binary (its own scaling profile)
    main.go
internal/
  config/                  # env-var config, one loader per binary (LoadAPI, LoadOutboxRelay)
  domain/                  # core types (Greeting, OutboxMessage) — no framework deps
  service/                 # GreetingService — shared business logic seam
  repository/              # storage interfaces + Transactor (no concrete DB)
  sqlite/                  # SQLite implementation of the repositories + Transactor + migrations
  outbox/                  # outbox relay poll/dispatch loop
  metrics/                 # business-metrics port + OpenTelemetry implementation
  feature/                 # feature-flag port + config-backed implementation
  otel/                    # OpenTelemetry tracer/meter/logger provider setup
  httpserver/              # generic chi router + shared middleware (auth, feature gate)
docs/                      # generated Swagger/OpenAPI output
```

### Key patterns

- **Service layer.** `service.GreetingService` is the single home for greeting business
  rules. Any future binary (a UI, a worker) wires the same `NewGreetingService`
  constructor rather than duplicating logic.
- **Transactions via `repository.Transactor`.** `WithinTx(ctx, fn)` runs multiple
  repository writes atomically without leaking a driver type above the interface.
- **Outbox pattern.** `CreateGreeting` writes the greeting row and an outbox row in the
  same transaction. `cmd/outbox-relay` polls the outbox table and dispatches events —
  delivery is **at-least-once**, so any real dispatch target must be idempotent.
- **Manual constructor injection.** Each `main.go` wires its dependencies explicitly,
  top to bottom. No DI framework.
- **Interface + hand-written mocks** are the testing seam; tests use the standard
  library `testing` package only.

## Getting started

### Prerequisites

- Go 1.26+
- `swag` for Swagger generation: `make deps`

### Configuration (environment variables)

Config is read from the environment only (a local `.env` file is loaded automatically).

| Variable | Required | Default | Used by |
| --- | --- | --- | --- |
| `AZURE_ISSUER` | yes | — | api |
| `AZURE_AUDIENCE` | yes | — | api |
| `DATABASE_PATH` | yes | — | api, outbox-relay |
| `ADDR` | no | `:8080` | api |
| `OTEL_EXPORTER` | no | `stdout` (`stdout` or `otlp`) | api, outbox-relay |
| `OTEL_EXPORTER_ENDPOINT` | no | `localhost:18889` | api, outbox-relay |

### Running locally

The API and the outbox relay are **separate processes** — run both:

```sh
make run-api      # starts the HTTP API (generates Swagger docs first)
make run-relay    # starts the outbox relay (in a second terminal)
```

Swagger UI is then available at http://localhost:8080/swagger/index.html.

To collect traces/metrics/logs locally, run an OTLP collector (e.g. the .NET Aspire
dashboard) and set `OTEL_EXPORTER=otlp`:

```sh
make aspire       # OTLP collector + dashboard on :18888 / :18889
make aspire-stop
```

## HTTP endpoints

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/health/live` | none | Liveness probe |
| GET | `/health/ready` | none | Readiness probe (pings the database) |
| POST | `/api/v1/greetings` | Bearer | Create a greeting |
| GET | `/api/v1/greetings` | Bearer | List greetings |
| GET | `/api/v1/greetings/{id}` | Bearer | Get a greeting by ID |
| GET | `/swagger/*` | none | Swagger UI |

Protected routes require an Entra ID (Azure AD) bearer token; the token is verified
against `AZURE_ISSUER`/`AZURE_AUDIENCE`. Copy `.http/template.http` to
`.http/localhost.http` and fill in your tenant/app details to acquire a token.

## Observability

`internal/otel` sets up the OpenTelemetry tracer, meter, and logger providers, shared by
every binary. Logs are bridged to the OTel logger provider so records carry trace
context. Business metrics (`greetings.created`, `greetings.listed`, `greetings.viewed`)
are recorded through the `metrics.GreetingMetrics` port; HTTP-level observability comes
from the `otelhttp` middleware in `internal/httpserver`.

## Development

```sh
make build        # build all binaries into bin/
make test         # run all tests
make lint         # go vet + gofmt check
make fmt          # format all Go source
make swagger      # regenerate docs/ from handler annotations
make load-test    # run the k6 load test
make help         # list all targets
```

## Persistence

SQLite (via `modernc.org/sqlite`, no cgo) with separate read/write connection pools, WAL
mode, and file-based migrations embedded in `internal/sqlite/migrations`. Migrations run
automatically on startup. The SQLite adapter is a plug-in point — swapping in another
database means providing a new implementation of the `internal/repository` interfaces.
