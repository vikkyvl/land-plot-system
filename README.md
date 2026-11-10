# Land Plot System

A microservice backend for storing land plots as geographic polygons and calculating their area and perimeter. Users register, describe a plot by its corner coordinates, and get area figures and saved reports in square metres, ares, and hectares.

The system is written in Go and runs locally with Docker Compose.

## Architecture

```
                 ┌──────────────┐
  client ──────► │ api-gateway  │  :8080 (the only public API port)
                 └──────┬───────┘
                        │ REST, resolved through Consul, round-robin
        ┌───────────────┼──────────────────┐
        ▼               ▼                  ▼
 identity-service  land-plot-service ◄── report-service
        │               │          REST     │
        ▼               ▼                   ▼
   identity_db       plot_db            report_db      (PostgreSQL)
                        │                   ▲
                        └── RabbitMQ events ┘

 Consul: service registry + health checks
 RabbitMQ: durable plot events and report-side projections
```

| Component           | Responsibility                                                                                     |
|---------------------|----------------------------------------------------------------------------------------------------|
| `api-gateway`       | Single entry point. Routes requests by path prefix, validates JWTs, balances across healthy instances. |
| `identity-service`  | User registration, login, and profile. Hashes passwords with bcrypt and issues HS256 JWTs.          |
| `land-plot-service` | Plot CRUD for the owner, polygon validation, calculations, and plot event publication.             |
| `report-service`    | Builds immutable reports over REST and maintains plot summaries from RabbitMQ events.              |

Each service owns its own database and does not read another service's data directly.

### Service discovery and routing

Every Go service registers a unique instance with an HTTP health check in Consul when it starts. The gateway and `report-service` look up passing instances in Consul for each call, so new instances join and failed ones leave without configuration changes. The gateway spreads requests across instances in round-robin order and adds an `X-Served-By` response header that names the instance that handled the request.

| Path prefix              | Target service      | Auth required |
|--------------------------|---------------------|---------------|
| `/api/v1/auth/*`         | `identity-service`  | no            |
| `/api/v1/users/profile`  | `identity-service`  | yes           |
| `/api/v1/plots/*`        | `land-plot-service` | yes           |
| `/api/v1/reports/*`      | `report-service`    | yes           |
| `/api/v1/demo/chain`     | `report-service`    | no            |

### Security

- The gateway checks the bearer token on protected routes, removes any client-supplied `X-User-*` headers, and forwards trusted identity headers to the target service.
- Each service also verifies the JWT itself, so no service relies only on the gateway.
- Users can read and change only their own plots and reports.

### Area calculation

The plot service uses the local radii of curvature of the WGS 84 ellipsoid. Coordinates are centred on the polygon. At the mean latitude, latitudes are scaled by the meridian radius of curvature and longitudes by the radius of the parallel. The area is then the shoelace formula on that plane, and the perimeter is the sum of the edge lengths. Coordinates stay `float64` throughout, and the API returns area and perimeter at full precision.

The calculation sits behind the `AreaCalculator` interface (`LocalEllipsoidCalculator`), which `bootstrap` passes to the plot service. Polygon validation is shared and runs before any calculator. The service rejects a polygon that has:
- fewer than three points,
- coordinates out of range,
- duplicate points,
- self-intersections,
- zero area.

For plots up to a few kilometres across, the method itself adds almost no error: on a reference square it matches GeographicLib within 0.01 %. The real accuracy is limited by the input coordinates, because an error of d metres per point changes the area by about perimeter × d. The result is therefore not a cadastral or legal measurement.

Updating a plot replaces its geometry and increments `version`. Concurrent updates are detected with optimistic locking, and the losing request gets `409 Conflict`. Each report records the plot `version` it was built from.

## Project layout

```
.
├── docker-compose.yml          # full local environment
├── .env.example                # overridable settings
├── .golangci.yml               # formatter and linter configuration
├── Makefile                    # fmt / lint / test / vet / compose checks, demos
├── deploy/postgres/            # creates per-service databases and roles
├── scripts/demo.sh             # end-to-end demo against the running stack
├── scripts/demo_lr3.sh         # transactional outbox and saga demo
├── tools/stripcomments/        # comment check used by make lint
└── services/
    ├── api-gateway/
    ├── identity-service/
    ├── land-plot-service/
    └── report-service/
```

Each service is an independent Go module with an N-layered structure:

```
<service>/
├── main.go
└── internal/
    ├── handler/               # HTTP: router, endpoint handlers, DTOs, request errors
    │   ├── middleware/        # bearer token authentication
    │   └── respond/           # JSON responses, request decoding, error → HTTP mapping
    ├── service/               # use cases, saga steps, outbox relay, and the ports they depend on
    ├── domain/                # entities, invariants, calculations, domain errors
    ├── repository/postgres/   # PostgreSQL adapter: tables, transactions, outbox (schema.sql is applied on startup)
    ├── messaging/rabbitmq/    # outbox publisher and reconnecting consumer
    ├── client/httpclient/     # land-plot-service client (report)
    ├── discovery/consul/      # Consul registration and instance resolution
    ├── auth/                  # JWT verification (and issuing in identity)
    ├── config/                # typed configuration from environment variables
    └── bootstrap/             # composition root: wiring, start, graceful shutdown
```

Dependencies point inward: `handler` → `service` → `domain`. `domain` imports nothing from the project, and `service` does not depend on PostgreSQL, RabbitMQ or HTTP. A use case that must change several tables atomically asks the repository for a transaction (`InTransaction`) and works through the `PlotStore` port inside it. Adapters implement interfaces declared by their consumers, and only `bootstrap` knows the concrete types. Each layer has its own data model — DTOs in `handler`, rows in `repository`, messages in `messaging` — with mappers at the boundaries.

Not every service has every package. The gateway has no domain or database; its layers are `handler` (middleware chain) → `proxy` (routes, reverse proxy) → `balancer`. Code that is the same in several services, such as the Consul registry or config helpers, is copied rather than shared, so each service can be built and deployed on its own.

## Getting started

Requirements: Docker with the Compose plugin. Go 1.26+ is needed only to run the checks locally.

```sh
cp .env.example .env   # optional: override the defaults
docker compose up -d --build
curl http://localhost:8080/health
```

The stack publishes these ports on the host:

| URL                       | What                                       |
|---------------------------|--------------------------------------------|
| `http://localhost:8080`   | API gateway                                |
| `http://localhost:8500`   | Consul UI                                  |
| `http://localhost:15672`  | RabbitMQ management UI                     |

The business services and PostgreSQL are reachable only inside the Compose network.

### RabbitMQ web UI

Open [http://localhost:15672](http://localhost:15672) in a browser and sign in with the local development credentials:

| Field | Default value |
|-------|---------------|
| Username | `land_app` |
| Password | `rabbitmq_dev_password` |

The port and credentials can be changed through `RABBITMQ_MANAGEMENT_PORT`, `RABBITMQ_USER`, and `RABBITMQ_PASSWORD` in `.env`.

The **Exchanges** tab contains `land.events` and `land.events.dead`. The **Queues and Streams** tab contains `report.plot-events` and `report.plot-events.dead`, where message counts, consumers, bindings, acknowledgements, and dead-lettered messages can be inspected.

Stop the stack with `docker compose down`. Add `-v` to also delete the database volume.

## API

All requests go through the gateway at `http://localhost:8080`. Errors use one JSON shape:

```json
{ "code": "plot_not_found", "message": "plot not found", "request_id": "..." }
```

Each service keeps its errors in one catalog, `internal/domain/errors.go`, plus `service/errors.go` and `handler/errors.go` for dependency and request errors. An entry declares the error kind, the stable `code` and the client message:

```go
ErrPlotNotFound = NewError(KindNotFound, "plot_not_found",
	"plot not found")
```

The kind does not depend on the transport. The HTTP handler maps it to a status code in one place (`statusByKind` in `handler/error_mapper.go`), so adding an error means adding one catalog entry. Any error that is not in a catalog becomes `500 internal_error` without internal details. The gateway has no domain layer, so its catalog in `proxy/errors.go` holds HTTP statuses directly.

### Auth

| Method | Path                     | Description                          |
|--------|--------------------------|--------------------------------------|
| POST   | `/api/v1/auth/register`  | Create a user (`email`, `password`)  |
| POST   | `/api/v1/auth/login`     | Get an `access_token`                |
| GET    | `/api/v1/users/profile`  | Current user                         |

```sh
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"password123"}'

TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"password123"}' | jq -r .access_token)

curl http://localhost:8080/api/v1/users/profile -H "Authorization: Bearer $TOKEN"
```

### Plots

| Method | Path                  | Description                                          |
|--------|-----------------------|------------------------------------------------------|
| POST   | `/api/v1/plots`       | Create a plot (`name`, `points[]` with `lat`/`lon`)  |
| GET    | `/api/v1/plots`       | List the current user's plots                        |
| GET    | `/api/v1/plots/{id}`  | Get one plot                                         |
| PATCH  | `/api/v1/plots/{id}`  | Update `name` and/or `points`                        |

```sh
curl -X POST http://localhost:8080/api/v1/plots \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"North field","points":[
        {"lat":50.4500,"lon":30.5230},{"lat":50.4500,"lon":30.5240},
        {"lat":50.4510,"lon":30.5240},{"lat":50.4510,"lon":30.5230}]}'

curl -X PATCH http://localhost:8080/api/v1/plots/<plot_id> \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"North field (renamed)"}'
```

A plot response includes `area_m2`, `perimeter_m`, `version`, `status`, and timestamps. A new or changed plot is `PENDING` until `report-service` confirms its projection, then `CONFIRMED`. `PATCH` on a `PENDING` plot returns `409 plot_not_confirmed`. A cancelled plot is hidden: the list skips it and `GET` returns `404 plot_not_found`.

### Reports

| Method | Path                    | Description                         |
|--------|-------------------------|-------------------------------------|
| POST   | `/api/v1/reports`       | Create a report for a plot (`plot_id`) |
| GET    | `/api/v1/reports/{id}`  | Get a saved report                  |

```sh
curl -X POST http://localhost:8080/api/v1/reports \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"plot_id":"<plot_id>"}'
```

A report contains `area_m2`, `area_ares`, `area_hectares`, `perimeter_m`, and the `plot_version` it was built from. Reports are built only for `CONFIRMED` plots; otherwise the request returns `409 plot_not_confirmed`.

### Plot events

`land-plot-service` publishes persistent JSON events to the durable topic exchange `land.events`. The routing keys are `plot.created`, `plot.updated`, and `plot.cancelled`. The versioned contract contains:

```json
{
  "schema_version": 1,
  "event_id": "uuid",
  "event_type": "plot.created",
  "occurred_at": "2026-10-26T12:00:00Z",
  "plot_id": "uuid",
  "owner_id": "uuid",
  "name": "North field",
  "area_m2": 10000,
  "version": 1
}
```

`report-service` consumes the durable `report.plot-events` queue with manual acknowledgements. In one PostgreSQL transaction it records `event_id` in `processed_events` (inbox), inserts or updates `plot_summaries`, and writes a `plot.summary.applied` reply to its own outbox. Duplicate IDs have no effect, and an event with an older or equal plot version cannot overwrite a newer summary. `plot.cancelled` marks the summary with `cancelled_at` and a higher version, so a late `plot.created` cannot bring the plot back. Invalid events and errors that wrap `domain.ErrRejectedEvent` (the simulated write failure) are rejected without requeue and are retained in `report.plot-events.dead`, which compensates the saga. Any other error, such as a `report_db` timeout, is treated as temporary: the event goes back to the queue after one second.

### Transactional outbox

Neither service publishes to RabbitMQ inside a request. The plot or summary change and the event are written to the `outbox_events` table in the same PostgreSQL transaction, so both are saved or neither is. A relay in each service polls the table every `OUTBOX_POLL_INTERVAL`, locks unpublished rows with `FOR UPDATE SKIP LOCKED` (several instances never take the same row), and publishes them in order with publisher confirms and the `mandatory` flag. After the broker confirms a message the relay sets `published_at`. If publishing fails it stores `last_error` and retries on the next tick. Only a rejection of the message itself (no queue is bound for the routing key, or the broker returns nack) increases `attempts`; after `OUTBOX_MAX_ATTEMPTS` rejections the row gets `failed_at`, is skipped from then on, and is logged at `ERROR`. An unavailable broker never counts as an attempt, so events survive a RabbitMQ outage. Delivery is at least once, and the inbox removes duplicates. Every `OUTBOX_CLEANUP_INTERVAL` the relay deletes published rows older than `OUTBOX_RETENTION`.

While RabbitMQ is down, `POST` and `PATCH` still return `201`/`200` and the events wait in the outbox. Publishers connect lazily, and consumers reconnect on their own, so no service restarts when the broker does.

### Saga

Creating or changing a plot is a choreographed saga between `land-plot-service` and `report-service`. In one transaction `land-plot-service` saves the plot as `PENDING`, a `plot_sagas` row (`STARTED`, the previous plot state for an update, and a deadline of now + `SAGA_TIMEOUT`), and the event in the outbox.

- **Success.** `report-service` applies the event and replies with `plot.summary.applied`. `land-plot-service` marks the saga `COMPLETED` and the plot `CONFIRMED`.
- **Failure in the consumer.** When `report-service` cannot write the summary, it rejects the message and RabbitMQ moves it to `land.events.dead`. `land-plot-service` receives a copy in `land-plot.saga-events` and compensates with reason `dlq`.
- **Timeout.** Every 5 seconds `land-plot-service` compensates `STARTED` sagas whose deadline has passed, with reason `timeout`.

Compensation of a create marks the plot `CANCELLED` and publishes `plot.cancelled`. Compensation of an update restores the previous name, points, and area with a new version and publishes `plot.updated`. Confirmation and compensation both start with `UPDATE plot_sagas … WHERE status = 'STARTED'`, so a repeated DLQ message, a late reply, or a race between timeout and success changes nothing.

To demonstrate a consumer failure, `report-service` fails to write the summary of a created or updated plot whose name starts with `REPORT_FAIL_NAME_PREFIX` (`FAIL` in `docker-compose.yml`, disabled when empty).

Inspect the asynchronous projection, the outbox, and the sagas after creating or updating a plot:

```sh
docker compose exec postgres psql -U report_user -d report_db \
  -c 'SELECT plot_id, name, area_m2, version, cancelled_at FROM plot_summaries ORDER BY updated_at DESC;'
docker compose exec postgres psql -U plot_user -d plot_db \
  -c 'SELECT routing_key, published_at, attempts FROM outbox_events ORDER BY position DESC LIMIT 5;' \
  -c 'SELECT plot_id, kind, status, reason FROM plot_sagas ORDER BY created_at DESC LIMIT 5;'
```

### Request chain trace

`GET /api/v1/demo/chain?message=hello` goes through gateway → report-service → land-plot-service and returns the list of instances that handled the request. It is useful for checking discovery and inter-service calls.

## Scaling

Run several instances of a service and the gateway balances across them:

```sh
docker compose up -d --build --scale land-plot-service=3
for i in 1 2 3; do
  curl -s -o /dev/null -D - http://localhost:8080/api/v1/plots \
    -H "Authorization: Bearer $TOKEN" | grep X-Served-By
done
```

You can see each instance and its health check in the Consul UI.

Scaling `report-service` adds competing consumers on the same `report.plot-events` queue. Each event is delivered to one instance, and the `event_id` check makes redelivery harmless.

## End-to-end demo

The demo script builds the stack, runs three Land Plot replicas, checks authentication, calculations, synchronous calls, RabbitMQ projections, idempotency, Round Robin, and failover, then scales `report-service` to two replicas and shows from the logs that both consume events from the shared queue:

```sh
make demo
```

It requires `docker`, `curl`, and `jq`. The script creates isolated demonstration records and temporarily stops one Land Plot container, which it always starts again during cleanup.

The outbox and saga demo shows a confirmed plot, compensation after a consumer write failure (`FAIL` name) and after a timeout (stopped `report-service`), a compensated update, an event that waits in the outbox while RabbitMQ is stopped, and a temporary `report_db` failure (a locked `plot_summaries` table) that is retried instead of cancelling the saga. It ends by checking that no outbox row was abandoned:

```sh
make demo-lr3
```

It stops `report-service` for about 30 seconds and RabbitMQ for about 10 seconds and starts them again during cleanup. It reads `SAGA_TIMEOUT` and `MESSAGE_PROCESSING_TIMEOUT` (whole seconds, such as `30s`) to size its waits and the table lock.

## Configuration

All settings come from environment variables. `docker-compose.yml` sets development defaults, and `.env.example` lists everything you can override:

- public ports,
- timeouts (HTTP shutdown, Consul, database, upstream),
- JWT secret and TTL,
- bcrypt cost,
- PostgreSQL and RabbitMQ credentials, connection and message-processing timeouts, and consumer prefetch,
- outbox poll interval, saga timeout, and the name prefix that makes `report-service` fail on purpose.

Set a strong `JWT_SECRET` and real passwords for any environment other than local development.

## Development

Requires [golangci-lint](https://golangci-lint.run/) v2.4 or newer.

```sh
make fmt       # format: gofumpt, import groups, line length
make lint      # formatting, whitespace, code quality and no-comments checks
make lint-fix  # remove comments, apply formatters and automatic lint fixes
make strip-comments  # only remove comments (tool directives are kept)
make test      # tests only
make check     # lint, go test, go vet for every service, and a docker compose config check
```

The linter configuration lives in `.golangci.yml` and is shared by all modules. Besides the standard checks, it enforces:

- **Vertical whitespace** with `wsl_v5`: blank lines separate logical steps, an error check follows its assignment directly, and blocks have no empty first or last line.
- **Formatting** with `gofumpt`, `gci` (standard library, third-party and module imports in separate groups) and `golines` (120 characters).
- **Code quality** with `revive`, `gocritic`, `errorlint`, `gosec`, `noctx` and others, plus function size and complexity limits (`funlen`, `gocognit`, `cyclop`).
- **Layer boundaries** with `depguard`: `domain` imports only the standard library (no HTTP, JSON or SQL), `service` imports only `domain` and `uuid`, `handler` imports only `domain`, `service` and its own subpackages, and adapters never import the HTTP layer or the configuration. Breaking the N-layered structure fails `make lint`.
- **No comments** with `tools/stripcomments`, since no golangci-lint linter removes comments. Go code carries no comments at all: names, small functions and the README explain it. Only tool directives such as `//go:embed` stay. The `revive` rules that require doc comments are therefore turned off.
