# Phoenix Online — Ticket Inventory Service

Go + Gin + MongoDB microservice for ticket types, stock, holds, and sales.

## Run locally

1. Copy environment file: `cp .env.example .env` and adjust values.
2. Start MongoDB — from `deployments/`: `docker compose up -d mongo` (or use the full stack below).
3. From repo root:

```bash
go run ./cmd/server
```

## Environment variables

| Variable | Description |
|----------|-------------|
| `PORT` | HTTP listen port (default `8080`) |
| `MONGODB_URI` | MongoDB connection URI |
| `MONGODB_DATABASE` | Database name |
| `ENVIRONMENT` | e.g. `development`, `staging`, `production` |
| `SERVICE_NAME` | Value for structured logs (`service` field) |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` |
| `HOLD_TTL_MINUTES` | Hold duration before expiry (default `15`) |

## Health

`GET /health` — liveness/readiness style check.

## API contract

See [api/openapi.yaml](api/openapi.yaml). Handlers are scaffolded; reservation logic is to be implemented next.

## Docker

```bash
docker build -t ticket-inventory-service .
docker run --rm -p 8080:8080 --env-file .env ticket-inventory-service
```

### Compose (Mongo + service)

From the `deployments/` directory:

```bash
docker compose up --build
```

The app waits for MongoDB health before starting.
