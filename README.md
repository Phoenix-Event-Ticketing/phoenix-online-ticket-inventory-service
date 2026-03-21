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

## CI (GitHub Actions)

On `push` to `main` or `dev`, after tests, govulncheck, Trivy, and Sonar quality gate succeed, the workflow builds and pushes the image to Google Artifact Registry.

**Secret:** `GCP_SA_KEY` — service account JSON for pushing images.

**Repository variables:** `GCP_PROJECT_ID`, `GCP_REGION`, `GCP_ARTIFACT_REGISTRY` (Artifact Registry repository name), `GCP_IMAGE_NAME` (image name within that repository).

Tags pushed: full commit SHA; plus `dev-latest` on `dev` and `main-latest` on `main`.

### GitOps (platform config repo)

After the image push succeeds, the `gitops-update` job checks out the Kustomize platform config repository, sets each overlay’s `images[].newTag` to the **same commit SHA** tag that was pushed (via `yq`, so only `name` + `newTag` are kept—no redundant `newName`). Overlays: `apps/inventory-service/overlays/dev` for `dev`, `overlays/prod` for `main`. The base deployment and overlays should use the same registry path as your `GCP_*` variables (see below). Validates with `kustomize build`, then commits and pushes to `main` on the config repo.

**Secret:** `GITOPS_REPO_PAT` — personal access token (or fine-grained PAT) with **`contents: write`** on the platform config repository (e.g. classic PAT with `repo` scope for private repos).

**Repository variables:**

| Variable | Required | Description |
|----------|----------|-------------|
| `GITOPS_REPO` | Yes | GitHub repo in `owner/name` form (e.g. `Phoenix-Event-Ticketing/phoenix-online-platform-config`). |
| `GITOPS_KUSTOMIZE_IMAGE_NAME` | No | Must match `images[].name` in each overlay (same as base deployment `image` without tag). If unset, defaults to the Artifact Registry image built from `GCP_REGION`, `GCP_PROJECT_ID`, `GCP_ARTIFACT_REGISTRY`, and `GCP_IMAGE_NAME`. Set this only if the config repo hardcodes a different registry path than those variables. |
