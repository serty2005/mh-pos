# MyHoReCa Cloud Backend

Minimal Cloud Sync Receiver for the POS/RMS platform.

Current scope:

- Go HTTP entrypoint: `cmd/cloud-api`
- PostgreSQL bootstrap and migrations
- `GET /health`
- `POST /api/v1/sync/edge-events`
- idempotent receive of POS Edge `SyncEnvelope`
- raw envelope storage before future projections

## Run

```powershell
cd cloud-backend
$env:CLOUD_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/mh_pos_cloud?sslmode=disable"
go run ./cmd/cloud-api
```

Defaults:

```text
CLOUD_HTTP_ADDR=:8090
CLOUD_POSTGRES_MIGRATIONS_DIR=migrations/postgres
```

`CLOUD_POSTGRES_DSN` is required.

## Test

```powershell
cd cloud-backend
go test ./...
```

The default tests use the in-memory repository for service and HTTP replay checks. PostgreSQL runtime storage is implemented in `internal/cloudsync/infra/postgres` and initialized by `migrations/postgres`.

## Contract

See `../docs/sync/edge-cloud-contracts-v1.md`.
