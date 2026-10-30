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
├── Makefile                    # fmt / lint / test / vet / compose checks, demo
├── deploy/postgres/            # creates per-service databases and roles
├── scripts/demo.sh             # end-to-end demo against the running stack
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
    ├── service/               # use cases and ports (interfaces) they depend on
    ├── domain/                # entities, invariants, calculations, domain errors
    ├── repository/postgres/   # PostgreSQL adapter (schema.sql is applied on startup)
    ├── messaging/rabbitmq/    # event publisher (land-plot) or consumer (report)
    ├── client/httpclient/     # land-plot-service client (report)
    ├── discovery/consul/      # Consul registration and instance resolution
    ├── auth/                  # JWT verification (and issuing in identity)
    ├── config/                # typed configuration from environment variables
    └── bootstrap/             # composition root: wiring, start, graceful shutdown
```

Dependencies point inward: `handler` → `service` → `domain`. `domain` imports nothing from the project, and `service` does not depend on PostgreSQL, RabbitMQ or HTTP. Adapters implement interfaces declared by their consumers, and only `bootstrap` knows the concrete types. Each layer has its own data model — DTOs in `handler`, rows in `repository`, messages in `messaging` — with mappers at the boundaries.

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

A plot response includes `area_m2`, `perimeter_m`, `version`, and timestamps.

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

A report contains `area_m2`, `area_ares`, `area_hectares`, `perimeter_m`, and the `plot_version` it was built from.

### Plot events

After committing a new or changed plot, `land-plot-service` publishes a persistent JSON event to the durable topic exchange `land.events`. The routing keys are `plot.created` and `plot.updated`. The versioned contract contains:

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

`report-service` consumes the durable `report.plot-events` queue with manual acknowledgements. In one PostgreSQL transaction it records `event_id` in `processed_events` and inserts or updates `plot_summaries`. Duplicate IDs have no effect, and an event with an older or equal plot version cannot overwrite a newer summary. Invalid events and database failures are rejected without requeue and are retained in `report.plot-events.dead`.

The producer uses publish-after-commit. RabbitMQ publisher confirms detect an immediate publication failure, but a process crash between the PostgreSQL commit and publish can still lose an event. A transactional outbox would close that gap and is not implemented.

If the broker does not confirm the event, `POST` and `PATCH` still succeed because the plot change is already saved. The service logs the failure at `ERROR` level with `plot_id`, `event_type`, and `error`, and the report projection stays behind until the plot changes again. Returning an error instead would invite a retry, and a retried `POST` would create a second plot.

Inspect the asynchronous projection after creating or updating a plot:

```sh
docker compose exec postgres psql -U report_user -d report_db \
  -c 'SELECT plot_id, name, area_m2, version FROM plot_summaries ORDER BY updated_at DESC;'
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

## Configuration

All settings come from environment variables. `docker-compose.yml` sets development defaults, and `.env.example` lists everything you can override:

- public ports,
- timeouts (HTTP shutdown, Consul, database, upstream),
- JWT secret and TTL,
- bcrypt cost,
- PostgreSQL and RabbitMQ credentials, connection and message-processing timeouts, and consumer prefetch.

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
