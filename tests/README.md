# Tests

## Integration (`integration/`)

These tests exercise the HTTP API against a real MongoDB instance.

### Prerequisites

- MongoDB reachable at `MONGODB_URI` (default `mongodb://127.0.0.1:27017` if unset).
- Start Mongo locally, for example from the repo root:

  ```bash
  docker compose -f deployments/docker-compose.yml up -d mongo
  ```

### Commands

From the repository root:

```bash
# Full suite including integration (requires Mongo)
go test -v ./...

# Skip integration tests (no Mongo needed)
go test -short ./...

# Only integration package
go test -v ./tests/integration/...
```

Each test run uses a **fresh database** named with a unique prefix and drops it in cleanup.

### CI

GitHub Actions starts a MongoDB service and sets `MONGODB_URI` before `go test ./...` so integration tests run on every push/PR (see `.github/workflows/ci.yml`).
