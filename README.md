# Land Plot System

A microservice system for storing land plots as geographic polygons and calculating their area and perimeter. Users sign in through Keycloak (also with a Google account), draw a plot on a map or send its corner coordinates, and get area figures and saved reports in square metres, ares, and hectares.

The backend is written in Go, the web app Arqetra (`apps/web`) in Vue 3 and TypeScript, and everything runs locally with Docker Compose.

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

 Keycloak (:8180): OAuth 2.0 / OpenID Connect server that issues all access tokens
 Consul: service registry + health checks
 RabbitMQ: durable plot events and report-side projections
```

| Component           | Responsibility                                                                                     |
|---------------------|----------------------------------------------------------------------------------------------------|
| `api-gateway`       | Single entry point. Applies CORS, verifies Keycloak tokens and scopes, routes by path prefix, balances across healthy instances. |
| `identity-service`  | Profile of the signed-in user, created from the access token on the first request.                 |
| `land-plot-service` | Plot CRUD for the owner, polygon validation, calculations, and plot event publication.             |
| `report-service`    | Builds immutable reports over REST and maintains plot summaries from RabbitMQ events.              |

Each service owns its own database and does not read another service's data directly.

The web app `apps/web` (Arqetra) is a client of the gateway like any other: an installable PWA served by nginx on `http://localhost:4173` that signs users in through Keycloak and draws plots on a Leaflet map. It calculates nothing itself: area, perimeter and validation come from the server (see [`apps/web/README.md`](apps/web/README.md)).

The `ai/` module adds AI access to the same API (see [`ai/README.md`](ai/README.md)). It is a separate Go module and works only through the gateway, like any other client:

| Component        | Responsibility                                                                                        |
|------------------|-------------------------------------------------------------------------------------------------------|
| `fc-client`      | Console LLM client: Gemini function calling over the REST API with token and prompt injection guardrails. |
| `mcp-server`     | MCP server with the plot and report tools, resources and prompts over stdio and Streamable HTTP.       |
| `producer-agent` | A2A agent over `land-plot-service`: creates and updates plots and waits for the saga.                  |
| `consumer-agent` | A2A agent over `report-service`: builds reports and delegates plot creation to the producer over A2A.  |

### Service discovery and routing

Every Go service registers a unique instance with an HTTP health check in Consul when it starts. The gateway and `report-service` look up passing instances in Consul for each call, so new instances join and failed ones leave without configuration changes. The gateway spreads requests across instances in round-robin order and adds an `X-Served-By` response header that names the instance that handled the request.

| Path prefix              | Target service      | Auth required |
|--------------------------|---------------------|---------------|
| `/api/v1/users/profile`  | `identity-service`  | yes           |
| `POST /api/v1/plots/preview` | `land-plot-service` | no, limited per IP |
| `/api/v1/plots/*`        | `land-plot-service` | yes           |
| `/api/v1/reports/*`      | `report-service`    | yes           |
| `/api/v1/demo/chain`     | `report-service`    | yes           |

### Security

- Keycloak issues all tokens; the gateway and every service verify them through the Keycloak JWKS (RS256) and check the scope of each route, so no service relies only on the gateway (Zero Trust).
- The gateway removes any client-supplied `X-User-*` headers and forwards trusted identity headers to the target service.
- The gateway applies a CORS policy that allows `https://cad.kpi.ua` and the local frontends.
- Users can read and change only their own plots and reports.

See [Authentication and authorization](#authentication-and-authorization) for the details.

### Area calculation

The plot service uses the local radii of curvature of the WGS 84 ellipsoid. Coordinates are centred on the polygon. At the mean latitude, latitudes are scaled by the meridian radius of curvature and longitudes by the radius of the parallel. The area is then the shoelace formula on that plane, and the perimeter is the sum of the edge lengths. Coordinates stay `float64` throughout, and the API returns area and perimeter at full precision.

The calculation sits behind the `AreaCalculator` interface (`LocalEllipsoidCalculator`), which `bootstrap` passes to the plot service. Polygon validation is shared and runs before any calculator. The service rejects a polygon that has:
- fewer than three points,
- coordinates out of range,
- duplicate points,
- self-intersections,
- zero area,
- more than 1000 points.

For plots up to a few kilometres across, the method itself adds almost no error: on a reference square it matches GeographicLib within 0.01 %. The real accuracy is limited by the input coordinates, because an error of d metres per point changes the area by about perimeter × d. The result is therefore not a cadastral or legal measurement.

### Area accuracy

Every area comes with its margin, because points placed on a map are never exact. The plot service estimates it in the domain (`AccuracyModel` in `internal/domain/accuracy.go`) from two optional facts the client sends with the points:

- `capture`: `{"zoom": 18, "layer": "map" | "satellite", "pointer": "mouse" | "touch" | "pen"}`;
- `obstructions`: `[{"side": 1, "reason": "trees" | "buildings" | "unclear"}]`, where `side` is the side from point `side` to point `side + 1`, counted from 0.

The error of one point is `σ_i = √(σ_click² + σ_obst²)` metres on each axis:

- `σ_click = k · r`, where `r = 156543.03 · cos φ / 2^zoom` is the ground size of a screen pixel at the mean latitude `φ`, and `k` is the click size in pixels (a pen counts as a mouse);
- `σ_obst` comes from the reason of a marked side and applies to both of its end points; a point between two marked sides takes the larger one.

The variance of the area is `σ_A² = ¼ · Σ σ_i² · |P(i+1) − P(i−1)|²` on the same local plane as the area, and the API reports `± 2σ_A` (about 95 %). For a square with side `L` and the same `σ` everywhere this gives `σ_A = √2 · σ · L`.

The map layer error `σ_geo` is not in `σ_A`: a shifted layer moves every point the same way, which does not change the area. It is reported apart as `ground_shift_m`, the possible shift of the whole boundary on the ground.

Without `capture` the service assumes the worst case (zoom 16, satellite, touch). The web app sends as `capture.zoom` the smallest zoom at which any of the current points was placed or moved, not the zoom of the last change, so zooming in at the end cannot make the margin smaller.

Example: a square plot of 2.48 ha placed at zoom 18 near Kyiv has a margin of ± 0.05 ha with a mouse and ± 0.13 ha with touch.

The constants are assumptions, set in the environment of `land-plot-service`:

| Variable | Default | Meaning |
|----------|---------|---------|
| `ACCURACY_MOUSE_CLICK_PX` | `3` | click size with a mouse or pen, pixels |
| `ACCURACY_TOUCH_CLICK_PX` | `8` | click size with a finger, pixels |
| `ACCURACY_MAP_SHIFT_M` | `2` | possible shift of the map layer, metres |
| `ACCURACY_SATELLITE_SHIFT_M` | `4` | possible shift of the satellite layer, metres |
| `ACCURACY_TREES_M` | `3` | boundary hidden by trees, metres |
| `ACCURACY_BUILDINGS_M` | `2` | boundary hidden by buildings or shadows, metres |
| `ACCURACY_UNCLEAR_M` | `5` | boundary hard to see, metres |
| `ACCURACY_FALLBACK_ZOOM`, `ACCURACY_FALLBACK_LAYER`, `ACCURACY_FALLBACK_POINTER` | `16`, `satellite`, `touch` | capture assumed when the client sends none |

The margin is calculated when a plot or version is read, from its points, `capture` and `obstructions`, so it is not stored and follows the constants.

Updating a plot replaces its geometry and increments `version`. Concurrent updates are detected with optimistic locking, and the losing request gets `409 Conflict`. Each report records the plot `version` it was built from.

## Project layout

```
.
├── docker-compose.yml          # full local environment
├── .env.example                # overridable settings
├── .golangci.yml               # formatter and linter configuration
├── Makefile                    # fmt / lint / test / vet / compose checks, demos
├── deploy/postgres/            # creates per-service databases and roles
├── infra/keycloak/            # Keycloak realm land-plot (imported on the first start) and the Arqetra login theme
├── postman/                   # Postman collection with OAuth 2.0 Authorization Code + PKCE
├── scripts/demo.sh             # end-to-end demo against the running stack
├── scripts/demo_lr3.sh         # transactional outbox and saga demo
├── scripts/demo_lr4.sh         # function calling, MCP and A2A demo
├── scripts/demo_lr5.sh         # CORS, Keycloak tokens, scopes and Zero Trust demo
├── scripts/demo_web.sh         # preview, deletion, history, report list, Mailpit and rate limit demo
├── scripts/keycloak_idp.sh     # adds the Google login to Keycloak
├── scripts/keycloak_realm.sh   # applies realm settings to a running Keycloak
├── tools/stripcomments/        # comment check used by make lint
├── ai/                         # AI agents module: fc-client, MCP server, A2A agents
├── apps/web/                   # Arqetra web app: Vue 3, PWA, nginx image
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
    │   ├── middleware/        # bearer token authentication and route scopes
    │   └── respond/           # JSON responses, request decoding, error → HTTP mapping
    ├── service/               # use cases, saga steps, outbox relay, and the ports they depend on
    ├── domain/                # entities, invariants, calculations, domain errors
    ├── repository/postgres/   # PostgreSQL adapter: tables, transactions, outbox (schema.sql is applied on startup)
    ├── messaging/rabbitmq/    # outbox publisher and reconnecting consumer
    ├── client/httpclient/     # land-plot-service client (report)
    ├── discovery/consul/      # Consul registration and instance resolution
    ├── auth/                  # Keycloak access token verification through JWKS
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
./scripts/keycloak_idp.sh   # optional: Google login, see "Google login"
curl http://localhost:8080/health
```

Moving from an earlier lab to Lab 5 needs empty volumes, because tokens and users now come from Keycloak: run `docker compose --profile ai down -v` first.

The stack publishes these ports on the host:

| URL                       | What                                       |
|---------------------------|--------------------------------------------|
| `http://localhost:8080`   | API gateway                                |
| `http://localhost:4173`   | Arqetra web app                            |
| `http://localhost:8180`   | Keycloak (login pages, tokens, admin console) |
| `http://localhost:8025`   | Mailpit: emails sent by Keycloak           |
| `http://localhost:8500`   | Consul UI                                  |
| `http://localhost:15672`  | RabbitMQ management UI                     |

The business services are reachable only inside the Compose network. PostgreSQL listens on `127.0.0.1:55432` for the integration tests.

The AI services run only with the `ai` profile, so the other demos need no LLM key:

```sh
docker compose --profile ai up -d --build
```

| URL                            | What                         |
|--------------------------------|------------------------------|
| `http://127.0.0.1:8090/mcp`    | MCP server (bearer token)    |
| `http://127.0.0.1:8091`        | A2A producer agent           |
| `http://127.0.0.1:8092`        | A2A consumer agent           |

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

Validation errors may add `details` with zero-based point indices, so a client can show which sides cross or which points are wrong:

```json
{ "code": "self_intersection", "message": "polygon must not intersect itself", "request_id": "...",
  "details": { "segments": [[1, 2], [3, 0]], "swap": [2, 3] } }
```

`duplicate_point` adds `{"indices": [i, j]}` and `invalid_coordinate` adds `{"index": i}`. Clients that do not know `details` can ignore it.

| HTTP | Codes |
|------|-------|
| 400 | `invalid_request`, `no_changes`, `invalid_plot_id`, `invalid_report_id`, `invalid_query`, `invalid_capture`, `invalid_obstruction` |
| 401 | `unauthorized` |
| 403 | `forbidden`, `insufficient_scope`, `origin_not_allowed` |
| 404 | `plot_not_found`, `report_not_found`, `route_not_found` |
| 409 | `version_conflict`, `plot_not_confirmed` |
| 410 | `plot_deleted` |
| 422 | `invalid_name`, `insufficient_points`, `invalid_coordinate`, `duplicate_point`, `self_intersection`, `zero_area`, `too_many_points` |
| 429 | `rate_limited` (with `Retry-After`) |
| 500, 502, 503 | `internal_error`, `bad_gateway`, `service_unavailable`, `plot_service_unavailable` |

Each service keeps its errors in one catalog, `internal/domain/errors.go`, plus `service/errors.go` and `handler/errors.go` for dependency and request errors. An entry declares the error kind, the stable `code` and the client message:

```go
ErrPlotNotFound = NewError(KindNotFound, "plot_not_found",
	"plot not found")
```

The kind does not depend on the transport. The HTTP handler maps it to a status code in one place (`statusByKind` in `handler/error_mapper.go`), so adding an error means adding one catalog entry. Any error that is not in a catalog becomes `500 internal_error` without internal details. The gateway has no domain layer, so its catalog in `proxy/errors.go` holds HTTP statuses directly.

### Profile

| Method | Path                     | Description                                                   |
|--------|--------------------------|---------------------------------------------------------------|
| GET    | `/api/v1/users/profile`  | Current user: `sub`, `email`, `name` and the token `scopes`   |

Tokens come from Keycloak; see [Getting a token in Postman](#getting-a-token-in-postman) for the browser flow and a `curl` example.

```sh
curl http://localhost:8080/api/v1/users/profile -H "Authorization: Bearer $TOKEN"
```

### Plots

| Method | Path                  | Description                                          |
|--------|-----------------------|------------------------------------------------------|
| POST   | `/api/v1/plots`       | Create a plot (`name`, `points[]` with `lat`/`lon`, optional `capture` and `obstructions`) |
| POST   | `/api/v1/plots/preview` | Measure and validate `points[]` without saving; no token needed |
| GET    | `/api/v1/plots`       | List the current user's plots, recently changed first |
| GET    | `/api/v1/plots/{id}`  | Get one plot                                         |
| GET    | `/api/v1/plots/{id}/versions` | Confirmed versions of a plot, newest first    |
| PATCH  | `/api/v1/plots/{id}`  | Update `name`, `points`, `capture` and `obstructions` |
| DELETE | `/api/v1/plots/{id}`  | Delete a confirmed plot (`204`)                      |

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

A plot response includes `area_m2`, `area_ares`, `area_hectares`, `perimeter_m`, `capture` (or `null`), `obstructions`, `accuracy`, `version`, `status`, and timestamps. `accuracy` is

```json
{
  "area_m2_margin": 520.4,
  "area_hectares_margin": 0.052,
  "percent": 2.1,
  "confidence": 0.95,
  "contributions": { "scale": 0.4, "obstructions": 0.6 },
  "worst_side": 1,
  "ground_shift_m": 4
}
```

with the shares of `σ_A²` per source and `worst_side` as the marked side that adds the most, or `null` without marks (see [Area accuracy](#area-accuracy)).

`PATCH` with `points` takes `capture` and `obstructions` from the same request: without them the worst case and no marks apply. `PATCH` with only `obstructions` replaces the marks of the current boundary and keeps its `capture`; a `PATCH` with only `name` keeps both. `capture` without `points` is `400 invalid_capture`. A new or changed plot is `PENDING` until `report-service` confirms its projection, then `CONFIRMED`. `PATCH` and `DELETE` on a `PENDING` plot return `409 plot_not_confirmed`. A cancelled plot is hidden: the list skips it and `GET` returns `404 plot_not_found`.

**Preview.** `POST /api/v1/plots/preview` takes `{"points": [...]}` with optional `capture` and `obstructions` and answers `200` with `{area_m2, area_ares, area_hectares, perimeter_m, accuracy, valid, errors[]}`; for an invalid polygon the numbers and `accuracy` are `null` and `errors` lists every problem with its `details`. It uses the same validation as `POST /api/v1/plots`, saves nothing, calls no external service and needs no token. An invalid `capture` or `obstructions` gives `400 invalid_capture` or `400 invalid_obstruction` (with `{"index": i}`). The gateway allows `PREVIEW_RATE_LIMIT_PER_MINUTE` requests per minute from one IP (60 by default) and answers `429 rate_limited` with `Retry-After` above that. On macOS all requests from the host reach the gateway from one Docker network address, so the limit is shared by the whole host.

**Deletion.** `DELETE` is soft: the plot gets `deleted_at`, a new version and the `plot.deleted` event through the outbox, without a saga. Afterwards `GET`, `PATCH`, `DELETE` and `/versions` return `410 plot_deleted` and the list no longer shows it.

**History.** A row in `plot_versions` is written every time a plot becomes `CONFIRMED`, also when a compensated update restores the previous boundary. Each version has `name`, `points`, `area_*`, `perimeter_m`, `capture`, `obstructions`, `accuracy` and `created_at`.

### Reports

| Method | Path                    | Description                         |
|--------|-------------------------|-------------------------------------|
| POST   | `/api/v1/reports`       | Create a report for a plot (`plot_id`) |
| GET    | `/api/v1/reports`       | List the current user's reports, newest first; `plot_id` and `limit` (50 by default, at most 100) are optional |
| GET    | `/api/v1/reports/{id}`  | Get a saved report                  |

```sh
curl -X POST http://localhost:8080/api/v1/reports \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"plot_id":"<plot_id>"}'
```

A report contains `area_m2`, `area_ares`, `area_hectares`, `perimeter_m`, the `plot_version` it was built from and `plot_deleted`. Reports are built only for `CONFIRMED` plots; otherwise the request returns `409 plot_not_confirmed`, and for a deleted plot `410 plot_deleted`. A wrong `limit` or `plot_id` in the list query gives `400 invalid_query`. When a plot is deleted, its reports stay and are marked with `plot_deleted: true`.

### Plot events

`land-plot-service` publishes persistent JSON events to the durable topic exchange `land.events`. The routing keys are `plot.created`, `plot.updated`, `plot.cancelled`, and `plot.deleted`. The versioned contract contains:

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

`GET /api/v1/demo/chain?message=hello` (any valid token) goes through gateway → report-service → land-plot-service and returns the list of instances that handled the request. It is useful for checking discovery and inter-service calls.

## Authentication and authorization

Users and programs get tokens from **Keycloak**, an OAuth 2.0 and OpenID Connect authorization server. The system issues no tokens of its own: `identity-service` has no passwords, no registration and no login.

### Keycloak

Keycloak runs in Docker Compose on `http://localhost:8180` and keeps its data in the `keycloak_db` database of the shared PostgreSQL. The realm `land-plot` is configuration as code: it is imported on the first start from `infra/keycloak/realm-land-plot.json`, which holds no secrets. Passwords and the client secret come from the environment (`${OLENA_KOVAL_PASSWORD}`, `${TARAS_MELNYK_PASSWORD}`, `${AI_CLIENT_SECRET}`).

The admin console is at `http://localhost:8180/admin` (`KC_BOOTSTRAP_ADMIN_USERNAME` / `KC_BOOTSTRAP_ADMIN_PASSWORD`, `admin` / `keycloak_admin_dev_password` by default).

The realm allows self-registration and password reset, and its pages are in Ukrainian by default (`uk`, with `en`). Keycloak sends its emails, such as the password reset link, to the development mail catcher **Mailpit**: nothing leaves the machine, and the messages are at `http://localhost:8025`.

The login pages use the **Arqetra theme** from `infra/keycloak/themes/arqetra/login`, mounted read-only into the container: sign-in, registration, password reset, the new password after the email, one-time password setup and entry, the error page and the expired page look like the web app. The theme extends `keycloak.v2`; it changes the look through its own CSS (the design tokens of `apps/web`, a local Montserrat font) and Ukrainian texts in `messages_uk.properties` with English as the fallback, and overrides only `login.ftl` and `register.ftl` (sign-in and register tabs) and `footer.ftl`. `start-dev` does not cache themes, so changes appear after a page reload. A test in `apps/web/tests/keycloakTheme.test.ts` keeps the theme tokens equal to the app tokens.

Keycloak imports the realm only when it does not exist yet. `./scripts/keycloak_realm.sh` applies the display name, login theme, registration, password reset, language and SMTP settings from the realm file to a running Keycloak without deleting volumes; it can be run any number of times and prints no passwords or secrets. To start from a clean realm, recreate the volumes:

```sh
docker compose --profile ai down -v
docker compose --profile ai up -d --build
./scripts/keycloak_idp.sh   # adds the Google login again
```

**Moving to Lab 5.** Earlier versions issued HS256 tokens from `identity-service`, and their users and plots do not exist in Keycloak. Start Lab 5 from empty volumes with `docker compose --profile ai down -v`; old data is not migrated.

### Clients

| Client ID           | Type         | Flow                                       | Used by                                                     |
|---------------------|--------------|--------------------------------------------|-------------------------------------------------------------|
| `land-plot-web`     | public       | Authorization Code with PKCE (S256 required) | the web app `apps/web` and Postman                        |
| `land-plot-ai`      | confidential | Client Credentials (service account)       | `fc-client`, `mcp-server`, `producer-agent`, `consumer-agent` |
| `land-plot-dev-cli` | public       | Direct Access Grants (password)            | the `demo*.sh` scripts only                                 |

`land-plot-web` accepts the redirect URIs `https://oauth.pstmn.io/v1/callback`, `http://localhost:5173/auth/callback` and `http://localhost:4173/auth/callback`, and the web origins `https://cad.kpi.ua`, `http://localhost:5173` and `http://localhost:4173`. Direct Access Grants are disabled on it.

`land-plot-dev-cli` exists only so that the demo scripts can get a token without a browser. **In production this client is disabled**, because the password grant hands user passwords to the client.

The realm has two test users: `olena.koval@example.com` and `taras.melnyk@example.com` (passwords from `.env`). `olena.koval` must set up a one-time password on the first login, which shows multi-factor authentication; the password grant cannot sign her in until that is done, so the scripts use `taras.melnyk`.

### Scopes

All access tokens carry the audience `aud = land-plot-api` (the default client scope `land-plot-audience`). The API scopes are optional client scopes, so a client gets only the scopes it asks for:

| Scope           | Allows                                |
|-----------------|---------------------------------------|
| `plots:read`    | read your plots                       |
| `plots:write`   | create and change your plots          |
| `reports:read`  | read your reports                     |
| `reports:write` | create reports                        |

| Method and path                                          | Required scope                     |
|----------------------------------------------------------|------------------------------------|
| `POST /api/v1/plots/preview`                             | none (public, rate limited)        |
| `GET /api/v1/plots`, `GET /api/v1/plots/{id}`, `GET /api/v1/plots/{id}/versions` | `plots:read` |
| `POST /api/v1/plots`, `PATCH /api/v1/plots/{id}`, `DELETE /api/v1/plots/{id}` | `plots:write` |
| `GET /api/v1/reports`, `GET /api/v1/reports/{id}`        | `reports:read`                     |
| `POST /api/v1/reports`                                   | `reports:write` and `plots:read`   |
| `GET /api/v1/users/profile`                              | `openid`                           |
| `GET /api/v1/demo/chain`                                 | any valid token                    |

`POST /api/v1/reports` also needs `plots:read` because `report-service` forwards the user's token to `land-plot-service` (`GET /internal/plots/{id}`, which requires `plots:read`) to read the plot. If `land-plot-service` refuses, the user gets `403`, not `502`.

The AI programs follow the principle of least privilege; Docker Compose gives each of them only what it needs through `AI_SCOPES`:

| Program                     | Scopes                                    |
|-----------------------------|-------------------------------------------|
| `producer-agent`            | `plots:read plots:write`                  |
| `consumer-agent`            | `plots:read reports:read reports:write`   |
| `fc-client`, `mcp-server`   | all four                                  |

### Token verification (Zero Trust)

The gateway **and** every service verify each token themselves; no service trusts the gateway alone, and a request sent directly to a service is checked the same way. Each module has its own `internal/auth.OIDCVerifier`:

- signing keys come from the JWKS of Keycloak (`OIDC_JWKS_URL`), are cached, refreshed every hour and, for an unknown `kid`, at most once every 30 seconds;
- only `RS256` and keys with `use=sig` are accepted, so `alg: none` and HS256 tokens are rejected;
- the signature, `iss` (`OIDC_ISSUER`), `aud` containing `land-plot-api`, a required `exp` and `nbf` are checked with 30 seconds of clock leeway;
- `sub` is required; `email` may be empty, as for a service account.

Browsers and Postman reach Keycloak at `http://localhost:8180`, containers at `http://keycloak:8080`. Keycloak runs with `KC_HOSTNAME=http://localhost:8180` and `KC_HOSTNAME_BACKCHANNEL_DYNAMIC=true`, so a token has the same `iss = http://localhost:8180/realms/land-plot` wherever it was requested, while containers load the JWKS and request tokens over the internal address.

Every service keeps its route-to-scope rules in one table (`protectedRoutes` in `handler/router.go`, `scopeRules` in the gateway `proxy/scopes.go`). Responses follow RFC 6750:

| Situation                       | Response                                                                                  |
|---------------------------------|-------------------------------------------------------------------------------------------|
| no token or an invalid token    | `401 unauthorized`, `WWW-Authenticate: Bearer error="invalid_token"`                       |
| a missing scope                 | `403 insufficient_scope`, `WWW-Authenticate: Bearer error="insufficient_scope", scope="plots:write"` |
| a method the gateway does not route | `404 route_not_found`                                                                  |

`identity-service` creates the user in `identity_db` from the token on the first `GET /api/v1/users/profile` (the user ID is the token `sub`) and updates the email and name on later requests.

### CORS

The gateway lets the department site `https://cad.kpi.ua` and the local frontends call the API from a browser. Its CORS middleware runs before authentication, so a preflight `OPTIONS` request without a token is answered with `204` instead of `401`.

- Allowed origins come from `CORS_ALLOWED_ORIGINS` (by default `https://cad.kpi.ua,http://localhost:5173,http://localhost:4173`) and are compared exactly; `*` is never used, and `https://cad.kpi.ua.evil.com` is not allowed.
- An allowed origin gets `Access-Control-Allow-Origin: <origin>`, `Vary: Origin` and `Access-Control-Expose-Headers: X-Request-ID, Retry-After`, so the browser can read the request ID and the wait after `429`. A preflight also gets `Allow-Methods: GET, POST, PATCH, DELETE, OPTIONS`, `Allow-Headers: Authorization, Content-Type, X-Request-ID`, `Max-Age: 600` and, when the browser asks for it, `Access-Control-Allow-Private-Network: true`, because a public page calls `localhost`.
- `Access-Control-Allow-Credentials` is not sent: the token travels in the `Authorization` header, not in a cookie.
- A foreign origin gets no `Access-Control-*` headers, and its preflight gets `403 origin_not_allowed`.
- The gateway removes `Access-Control-*` headers that a service might return (`ReverseProxy.ModifyResponse`), so each response carries exactly one set.

### Google login

The Keycloak login page has a **Google** button, and a frontend can skip the Keycloak page by adding `kc_idp_hint=google` to the authorization request.

1. In the Google Cloud Console open **APIs & Services → OAuth consent screen**, choose **External**, keep the app in **Testing** and add your Google account to **Test users**.
2. Open **Credentials → Create credentials → OAuth client ID**, choose **Web application** and add the authorized redirect URI `http://localhost:8180/realms/land-plot/broker/google/endpoint`.
3. Put the client ID and secret into `.env` as `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET`.
4. With Keycloak running, run `./scripts/keycloak_idp.sh`.

The script adds or updates the identity provider `google` through `kcadm.sh` inside the Keycloak container and is safe to run again; without the two variables it prints a warning and exits with `0`. The first Google login creates the user in Keycloak, the email from Google counts as verified, and the email and name are refreshed on each login. Open `http://localhost:8180/realms/land-plot/account` and press **Google** to check it.

### Getting a token in Postman

The collection `postman/lr5_collection.json` uses OAuth 2.0 on the collection level:

| Setting            | Value                                                                              |
|--------------------|------------------------------------------------------------------------------------|
| Grant type         | Authorization Code (With PKCE), code challenge method `SHA-256`                   |
| Auth URL           | `http://localhost:8180/realms/land-plot/protocol/openid-connect/auth`             |
| Access token URL   | `http://localhost:8180/realms/land-plot/protocol/openid-connect/token`            |
| Client ID          | `land-plot-web` (no secret, public client)                                         |
| Callback URL       | `https://oauth.pstmn.io/v1/callback` (authorize using the browser)                 |
| Scope              | `openid profile email plots:read plots:write reports:read reports:write`           |

Import the collection, open its **Authorization** tab, press **Get New Access Token**, sign in with Google or a test user and press **Use Token**. The requests then show the profile, list and create plots and build a report. The request **Create plot with plots:read only (403)** has its own OAuth settings with the scope `openid plots:read` and gets `403 insufficient_scope`.

Without a browser a token for a test user comes from the development client:

```sh
TOKEN=$(curl -s http://localhost:8180/realms/land-plot/protocol/openid-connect/token \
  -d grant_type=password -d client_id=land-plot-dev-cli \
  -d username=taras.melnyk@example.com -d password=TarasDevPassword123 \
  --data-urlencode 'scope=openid plots:read plots:write reports:read reports:write' | jq -r .access_token)

cut -d. -f2 <<<"$TOKEN" | tr '_-' '/+' | base64 -d 2>/dev/null | jq '{iss, aud, sub, scope, exp}'
```

### Production

The Compose setup is for local development. In production Keycloak runs with `start` instead of `start-dev`, behind TLS on its own hostname, `land-plot-dev-cli` and every password grant are disabled, the bootstrap admin is replaced by a real administrator account, and all secrets come from a secret store.

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

The AI demo checks the MCP server and both A2A agents without the LLM, then runs every LLM scenario once when `LLM_API_KEY` is set in `.env`. It also needs `MCP_ACCESS_TOKEN` and Go:

```sh
make demo-lr4
```

The security demo runs without a browser. It shows a decoded Keycloak token, a CORS preflight from `https://cad.kpi.ua` and from a foreign origin, exactly one `Access-Control-Allow-Origin` on a response, `401` with `WWW-Authenticate` without a token, `401` for a forged and an HS256 token, `403 insufficient_scope` for a token with `plots:read` only, a plot, saga and report with all scopes, `401` from `land-plot-service` called directly without a token, the profile created in `identity_db`, and `403` for the producer agent credentials on `POST /api/v1/reports`:

```sh
make demo-lr5
```

The web backend demo shows the preview with and without errors and its `details`, `422` with the same details on save, a changed boundary and its version history, the report list with `plot_id` and `limit`, deletion (`409` while pending, `204`, then `410` and a report marked `plot_deleted`), a CORS preflight for `DELETE`, a password reset email in Mailpit and the preview rate limit (`429` with `Retry-After`):

```sh
make demo-web
```

All demo scripts get tokens from Keycloak: the user `taras.melnyk@example.com` through `land-plot-dev-cli`, and the AI service account through client credentials.

## Configuration

All settings come from environment variables. `docker-compose.yml` sets development defaults, and `.env.example` lists everything you can override:

- public ports,
- timeouts (HTTP shutdown, Consul, database, upstream),
- Keycloak: port, database, bootstrap admin, test user passwords and the `land-plot-ai` client secret,
- token verification (`OIDC_ISSUER`, `OIDC_JWKS_URL`, `OIDC_AUDIENCE`, `OIDC_TIMEOUT`), the CORS origins (`CORS_ALLOWED_ORIGINS`) and the preview limit (`PREVIEW_RATE_LIMIT_PER_MINUTE`),
- the Mailpit web UI port (`MAILPIT_UI_PORT`),
- the web app container: port, the gateway and Keycloak addresses built into the bundle and the CSP origins (`WEB_*`, see [`apps/web/README.md`](apps/web/README.md)),
- the Google client (`GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, only in `.env`),
- PostgreSQL and RabbitMQ credentials, connection and message-processing timeouts, and consumer prefetch,
- outbox poll interval, saga timeout, and the name prefix that makes `report-service` fail on purpose,
- the LLM key and model, agent token and step limits, the AI client credentials and scopes, and the MCP access token (see [`ai/README.md`](ai/README.md)).

Set real passwords and secrets for any environment other than local development, and never commit `.env`.

## Development

Requires [golangci-lint](https://golangci-lint.run/) v2.4 or newer.

```sh
make fmt       # format: gofumpt, import groups, line length
make lint      # formatting, whitespace, code quality and no-comments checks
make lint-fix  # remove comments, apply formatters and automatic lint fixes
make strip-comments  # only remove comments (tool directives are kept)
make test      # tests only
make check     # lint, go test, go vet for every module, including ai/, and docker compose config checks
make web-check # web app: lint (with layer boundaries), tests, build
make check-all # make check and make web-check
make ai-mcp-inspector  # MCP Inspector with the MCP server over stdio
```

The linter configuration lives in `.golangci.yml` and is shared by all modules. Besides the standard checks, it enforces:

- **Vertical whitespace** with `wsl_v5`: blank lines separate logical steps, an error check follows its assignment directly, and blocks have no empty first or last line.
- **Formatting** with `gofumpt`, `gci` (standard library, third-party and module imports in separate groups) and `golines` (120 characters).
- **Code quality** with `revive`, `gocritic`, `errorlint`, `gosec`, `noctx` and others, plus function size and complexity limits (`funlen`, `gocognit`, `cyclop`).
- **Layer boundaries** with `depguard`: `domain` imports only the standard library (no HTTP, JSON or SQL), `service` imports only `domain` and `uuid`, `handler` imports only `domain`, `service` and its own subpackages, and adapters never import the HTTP layer or the configuration. Breaking the N-layered structure fails `make lint`.
- **No comments** with `tools/stripcomments`, since no golangci-lint linter removes comments. Go code carries no comments at all: names, small functions and the README explain it. Only tool directives such as `//go:embed` stay. The `revive` rules that require doc comments are therefore turned off.
