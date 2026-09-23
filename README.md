# JobHub AI

A modular job search aggregator designed to bring legal public job sources, resume matching, and personalized recommendations into one workspace.

**Current scope: Phase 1.5 — Supabase-ready database hosting.** Local development uses Docker PostgreSQL; staging and production can use Supabase Postgres. The Go API remains the application backend. Supabase Auth, Edge Functions, and a frontend Supabase client are deferred to Phase 2.

## Included

- React + TypeScript + Vite frontend, Tailwind CSS, React Router, Axios, and TanStack Query. Zustand is installed for future local state; no artificial store is needed yet.
- Go + Gin API with validated environment configuration, structured JSON logs, graceful shutdown, HTTP timeouts, explicit CORS origin, security headers, and consistent JSON errors.
- PostgreSQL 16, pgx connection pool, bounded database startup/readiness checks.
- Profile-based Docker Compose with a complete local PostgreSQL stack and an API-only profile for an external PostgreSQL URL.
- golang-migrate baseline and sqlc configuration with generated pgx/v5 code.
- Tests for configuration, health responses, database failure propagation, and context cancellation.

## Architecture and structure

```text
Browser → Vite /api proxy → Gin handler → HealthService → pgx pool → PostgreSQL
                                            ↑
                                  golang-migrate baseline

PostgreSQL = local Docker (development) or Supabase Postgres (staging/production)

jobhub-ai/                         # repository root (current directory)
├── frontend/
│   ├── public/
│   ├── src/
│   │   ├── api/                   # Axios client and health request
│   │   ├── components/
│   │   ├── features/
│   │   ├── hooks/
│   │   ├── layouts/
│   │   ├── pages/                 # foundation status screen
│   │   ├── store/
│   │   ├── types/
│   │   ├── utils/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   └── styles.css
│   ├── index.html
│   ├── package.json
│   ├── package-lock.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── backend/
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── config/
│   │   ├── database/dbgen/         # generated; do not hand-edit
│   │   ├── handlers/
│   │   ├── middleware/
│   │   ├── models/
│   │   ├── repositories/
│   │   ├── services/
│   │   ├── integrations/
│   │   ├── dto/
│   │   ├── utils/
│   │   └── validation/
│   ├── migrations/
│   ├── sql/queries/health.sql
│   ├── sqlc.yaml
│   ├── go.mod
│   ├── go.sum
│   ├── Dockerfile
│   └── .dockerignore
├── docker-compose.yml
├── Makefile
├── scripts/migrate-remote.sh
├── .env.example
├── .gitignore
└── README.md
```

Empty directories are tracked with `.gitkeep` for subsequent phases. Handlers depend on service interfaces; database access remains outside handlers. The baseline creates the `jobhub` schema only, leaving domain tables for later migrations. The generated timestamp query validates the sqlc toolchain; readiness uses pgx `Ping` directly.

## Quick start

Prerequisites: Docker with Compose v2, Node.js 22.12+ (or supported newer version), and npm. Go 1.25+ is needed only to develop the API locally.

From the repository root:

```sh
cp .env.example .env
make local-up
```

In another terminal:

```sh
cd frontend
npm ci
npm run dev
```

Open [the frontend](http://localhost:5173). The backend is at [the health endpoint](http://localhost:8080/api/v1/health).

The `local` profile starts PostgreSQL, waits for its health check, applies migrations, then starts the API. A successful migration container exit is expected. The frontend runs locally with Vite; Compose provides the API and database.

```sh
curl -i http://localhost:8080/api/v1/health
docker compose ps -a
docker compose --profile local logs backend-local migrate
```

Stop with `make local-down`. The named `postgres_data` volume survives container restarts and normal `down`. **`docker compose --profile local down -v` deletes local database data.**

## Configuration

Copy `.env.example`; never commit `.env`. Compose reads the root `.env` automatically. Vite reads it from the repository root but exposes only `VITE_` variables to client code. The Go process uses actual environment variables and does not implicitly read dotenv files.

| Variable | Purpose |
| --- | --- |
| `APP_ENV` | `development`, `test`, `staging`, or `production`; production disables Gin debug mode |
| `PORT` | API host port; also the listener port when running Go locally |
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | Compose database credentials and database name |
| `POSTGRES_PORT` | Local PostgreSQL published port, default 5432 |
| `DATABASE_URL` | Main runtime PostgreSQL connection string |
| `DATABASE_MIGRATION_URL` | Optional direct/session URL used only by migration commands; falls back to `DATABASE_URL` |
| `PGX_QUERY_EXEC_MODE` | `cache_statement` for direct/session; `exec` for transaction pooling; `simple_protocol` is an available fallback |
| `FRONTEND_ORIGIN` | Exact allowed browser origin, default `http://localhost:5173` |
| `VITE_API_PROXY_TARGET` | Vite server's API proxy destination, default `http://localhost:8080` |

The local Compose profile builds its container URL with hostname `postgres`; the example root URL uses `localhost` for a Go process running on the host. Both use `sslmode=disable`. Supabase URLs must use `sslmode=require`. URLs pin `search_path=public` so migration tracking stays in a stable schema. URL-encode reserved characters in passwords. Changing `POSTGRES_*` does not change accounts in an already initialized local volume.

If changing `PORT`, also update `VITE_API_PROXY_TARGET`. If changing the Vite port, update `FRONTEND_ORIGIN`. No JWT or provider secrets are needed before their respective phases. The example password is for local development only. Production deployment requires separate secret management, TLS, and deployment hardening.

## Run the API locally

Start infrastructure and apply migrations:

```sh
docker compose --profile local up -d postgres
make migrate-local
```

Export your trusted local configuration from the repository root, then run Go:

```sh
set -a
. ./.env
set +a
cd backend
go mod download
go run ./cmd/api
```

Do not run the Compose backend and local backend on the same port. The API fails fast when configuration or the database connection is invalid. It stops accepting requests and closes the pool on SIGINT/SIGTERM.

## Migrations

All schema changes belong in numbered `up`/`down` SQL migrations. Never edit a migration already deployed. The baseline has no user or job tables.

The local profile applies pending migrations on startup. For an explicit rerun:

```sh
make migrate-local
```

For local CLI use (run from `backend`, with `DATABASE_URL` exported):

```sh
go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.20.1
migrate -path migrations -database "$DATABASE_URL" up
migrate -path migrations -database "$DATABASE_URL" version
# Development rollback of one migration; review its effects first:
migrate -path migrations -database "$DATABASE_URL" down 1
# Create a later-phase migration:
migrate create -ext sql -dir migrations -seq create_users
```

The pinned migration CLI requires Go 1.25.11 or newer; Go automatically downloads a compatible toolchain when needed.

Ensure `$(go env GOPATH)/bin` is in your PATH after installing Go tools. The baseline rollback uses `RESTRICT` so it cannot silently delete populated application objects. If a migration fails and the database is marked dirty, inspect and repair the schema before setting any migration version; do not blindly force it.

For a remote staging database, put a direct or session-mode URL in the untracked root `.env` (or export it in the shell) and run:

```sh
export DATABASE_MIGRATION_URL='postgresql://postgres.<project-ref>:<PASSWORD>@<pooler-host>:5432/postgres?sslmode=require&search_path=public'
make migrate-staging
```

`scripts/migrate-remote.sh` uses the pinned migration image and refuses port `6543`. Schema migrations require a direct or session connection because transaction pooling does not preserve the session features migration tools may need. If `DATABASE_MIGRATION_URL` is unset, the script falls back to `DATABASE_URL`.

## Supabase setup

Create separate Supabase projects for staging and production. In each project's **Connect** dialog, copy the exact URL instead of constructing the pooler host yourself. Keep real URLs in your deployment platform's secret store or an untracked local `.env`; never put them in source control.

Choose the connection as follows:

| Connection | Runtime use | Port | pgx mode |
| --- | --- | --- | --- |
| Direct | Preferred for a persistent Go service when its network supports IPv6, or when the Supabase IPv4 add-on is enabled | `5432` | `cache_statement` |
| Shared pooler, session mode | Persistent Go service on an IPv4-only host | `5432` | `cache_statement` |
| Shared pooler, transaction mode | Highly scaled or short-lived runtimes only | `6543` | `exec` |

Supabase direct database hosts are IPv6 by default. The shared pooler is reachable over IPv4; use its session mode when the deployment platform cannot reach IPv6. Transaction mode does not support prepared statements or persistent session state, so `PGX_QUERY_EXEC_MODE=exec` disables pgx's prepared-statement cache while retaining the extended protocol. Migrations must still use a direct or session URL in `DATABASE_MIGRATION_URL`.

Example staging runtime configuration, using placeholders only:

```dotenv
APP_ENV=staging
DATABASE_URL=postgresql://postgres.<project-ref>:<PASSWORD>@<pooler-host>:5432/postgres?sslmode=require&search_path=public
DATABASE_MIGRATION_URL=postgresql://postgres:<PASSWORD>@db.<project-ref>.supabase.co:5432/postgres?sslmode=require&search_path=public
PGX_QUERY_EXEC_MODE=cache_statement
```

If direct IPv6 is unavailable to the machine running migrations, use the shared session pooler URL on port `5432` for `DATABASE_MIGRATION_URL`. After applying migrations, start an API container against the external database with:

```sh
docker compose --profile external up --build backend
curl -i http://localhost:8080/api/v1/health
```

The `jobhub` schema is not intended for the Supabase Data API. Do not add it to exposed schemas and do not grant its tables to `anon` or `authenticated`. The Go backend uses a server-side database connection, whose password must never reach browser code. Every future application-table migration must include `ALTER TABLE ... ENABLE ROW LEVEL SECURITY`; see [the migration security rules](backend/migrations/README.md). Explicit ownership policies can be added in Phase 2 if direct client access is selected.

Free Plan projects with low activity may be paused after a seven-day period. Resume a paused staging project in the Supabase dashboard before interpreting a failed health check as an application defect. Production should use a plan whose availability matches the service requirements.

## sqlc

From `backend`:

```sh
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate
```

Configuration uses version 2, PostgreSQL migrations as schema, and `pgx/v5` output. Commit generated files under `internal/database/dbgen` alongside query changes. No running database is needed to generate this query.

## API overview

| Method | Endpoint | Behavior |
| --- | --- | --- |
| GET | `/api/v1/health` | Checks the database with a two-second deadline |

Healthy response, HTTP 200:

```json
{"status":"ok","database":"up","service":"jobhub-ai"}
```

Unavailable database, HTTP 503:

```json
{"error":{"code":"DATABASE_UNAVAILABLE","message":"Database is unavailable"}}
```

Unknown routes and unsupported methods return structured 404/405 errors. Database internals are logged server-side, not returned in responses. Health is a readiness check: it calls `pgxpool.Ping` with a two-second deadline and reports 503 if either local PostgreSQL or Supabase becomes unavailable. Registration, job APIs, and protected routes do not exist yet.

## Checks

```sh
cd backend
go test -race ./...
go vet ./...
go build ./cmd/api
cd ../frontend
npm ci
npm run build
```

The frontend build includes strict TypeScript checking. Its status page handles loading, unavailable API/database, and retry without reporting simulated health. After starting Compose, verify the health endpoint and refresh the frontend status.

## Screenshots

Placeholder: add desktop/mobile screenshots of the foundation page after running the project. Later phases will include search, resume analysis, and dashboard screenshots.

## Next phases

Next, **Phase 2**: user migration and model, bcrypt password hashing, registration/login, expiring JWTs, current-user endpoint, and protected routes. Later phases add normalized providers and concurrent aggregation, search UI, legal public APIs, saved jobs/profile/history, private resume parsing, deterministic matching, recommendations, and expanded tests/security. No scraping is planned; unavailable integrations will use clearly labeled mocks.

## References

- [Vite setup and runtime requirements](https://vite.dev/guide/)
- [sqlc configuration](https://docs.sqlc.dev/en/latest/reference/config.html)
- [golang-migrate CLI](https://github.com/golang-migrate/migrate/blob/master/cmd/migrate/README.md)
- [Supabase database connections](https://supabase.com/docs/guides/database/connecting-to-postgres)
- [Supabase project pausing](https://supabase.com/docs/guides/platform/free-project-pausing)

## Verification in the development environment

Frontend production build, Go build, `go test -race ./...`, `go vet ./...`, and sqlc generation are the required code checks. The local Compose and Supabase runtime checks require Docker and, for Supabase, a user-provided connection string.
