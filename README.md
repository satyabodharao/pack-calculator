# Order Packs Calculator

A small Go application that calculates how many of each **pack size** to ship for
a customer order, shipping only **whole packs** while minimising waste.

Given configurable pack sizes (e.g. `250, 500, 1000, 2000, 5000`), the service
decides the packs to ship according to three rules, in strict priority order:

1. **Only whole packs** may be shipped — packs cannot be broken open.
2. Subject to Rule 1, ship the **least number of items** possible.
3. Subject to Rules 1 & 2, ship the **fewest number of packs** possible.

> Rule 2 always takes precedence over Rule 3.

The application exposes a **JSON HTTP API** and a **simple web UI**, is fully
**containerised**, and ships with a comprehensive **unit-test** suite.

---

## Table of contents

- [Architecture & layers](#architecture--layers)
- [API reference](#api-reference)
- [Build, run & test](#build-run--test)
- [Configuration](#configuration)
- [Deployment (Heroku)](#deployment-heroku)

---

## Architecture & layers

The code follows a clean, inward-pointing layered design. Each layer depends
only on the layer(s) beneath it, and the core algorithm has **zero dependencies**.

```
HTTP request 
  -> api  (internal/api) 
    -> service (internal/service) 
      -> calculator (internal/calculator)
      -> repository (internal/repository)
 

**Why layers?**
- The `calculator` is a pure function — trivial to test exhaustively, reusable
  anywhere, and immune to changes in transport or storage.
- The `repository` is an **interface**. Today it is backed by an in-memory store;
  swapping in a Postgres (or any other) backend means writing one new type that
  satisfies `PackSizeRepository` — **no changes** to the service or API layers.
- The `service` holds business rules and is independently testable without a
  running HTTP server.
- The `api` layer is thin: it only translates between HTTP and the service.
```

---

## API reference

Base URL (local): `http://localhost:8080`

### `GET /api/pack-sizes`
Returns the currently configured pack sizes.
```json
{ "pack_sizes": [250, 500, 1000, 2000, 5000] }
```

### `PUT /api/pack-sizes`
Replaces the configured pack sizes. Sizes must be positive integers.
```bash
curl -X PUT http://localhost:8080/api/pack-sizes \
  -H 'Content-Type: application/json' \
  -d '{"pack_sizes":[250,500,1000,2000,5000]}'
```

### `POST /api/calculate`
Calculates the packs to ship for an order.
```bash
curl -X POST http://localhost:8080/api/calculate \
  -H 'Content-Type: application/json' \
  -d '{"items":12001}'
```
```json
{
  "items": 12001,
  "packs": [
    { "size": 5000, "count": 2 },
    { "size": 2000, "count": 1 },
    { "size": 250,  "count": 1 }
  ],
  "total_items": 12250,
  "total_packs": 4
}
```

### `GET /healthz`
Liveness probe used by Docker/Heroku. Returns `{"status":"ok"}`.

### `GET /`
Serves the web UI.

**Error format** (any 4xx/5xx): `{ "error": "human-readable message" }`.

---

## Build, run & test

Requires **Go 1.26+** (for local builds) and/or **Docker** (for containerised run).

### Run with Docker (recommended)
```bash
docker compose up --build
# UI + API now at http://localhost:8080
```
Tear down with `docker compose down`.

### Run locally with Go
```bash
go run ./cmd/server
# UI + API at http://localhost:8080  (set PORT to change)
```

### Run the tests
```bash
go test ./... -race -cover
```
The suite covers:
- **Algorithm** — all brief examples, the `{23,31,53}@500000` stress case, edge
  cases (zero/negative order, exact multiples, single pack size, duplicates).
- **Repository** — seeding, read/write, defensive copying, concurrency (`-race`).
- **Service** — validation and orchestration.
- **API** — every endpoint via `httptest`: happy paths, bad JSON, wrong methods,
  validation errors.

### Testing the UI
1. Start the app (Docker or `go run`).
2. Open <http://localhost:8080>.
3. Under **Pack Sizes**, add/remove/edit sizes and click *Submit pack sizes
   change* — the saved set is echoed back.
4. Under **Calculate packs for order**, enter an item count and click
   *Calculate* — the Pack/Quantity table and totals update.

---

## Configuration

The server is configured via environment variables:

| Variable  | Default | Purpose                                         |
|-----------|---------|-------------------------------------------------|
| `PORT`    | `8080`  | TCP port to listen on (Heroku sets this).       |
| `WEB_DIR` | `web`   | Directory containing the static UI (`index.html`). |

Pack sizes themselves are **runtime-configurable** via the API/UI (no code or
config changes needed) and seeded with `250, 500, 1000, 2000, 5000`.

---

## Deployment (Heroku)

The app deploys to Heroku as a **container** (the `Dockerfile` is the source of
truth; `heroku.yml` wires it up). The server reads the `$PORT` Heroku provides.

```bash
# One-time setup
heroku login
heroku create <your-app-name>
heroku stack:set container -a <your-app-name>

# Deploy
git push heroku main           # if the Heroku remote is configured
# (or) heroku container:push web -a <your-app-name>
#      heroku container:release web -a <your-app-name>

# Open it
heroku open -a <your-app-name>
```

No database add-on is required — storage is in-memory (see notes below).
