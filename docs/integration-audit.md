# JobHub integration audit

Status: audited proposal, awaiting approval of the missing backend contracts.
This document describes repository evidence, not a deployed marketplace.

## Existing implementation

| Area | Evidence | Implemented behavior |
| --- | --- | --- |
| Routing | `backend/internal/handlers/router.go` | Only `GET /api/v1/health`; unknown routes return JSON 404 |
| Health | `backend/internal/handlers/health.go`, `services/health.go` | Anonymous readiness check, pgx Ping with a two-second deadline |
| Database | `backend/internal/database/database.go` | PostgreSQL pool, configurable pgx execution mode, connection URL-driven TLS |
| Schema | `backend/migrations/000001_initialize.up.sql` | Creates `jobhub` schema only; no domain tables |
| SQL | `backend/sql/queries/health.sql` | Timestamp query only |
| Domain models | `backend/internal/database/dbgen/models.go` | No generated domain structs |
| Middleware | `backend/internal/middleware/http.go` | Logging, response headers, exact-origin CORS; allowed methods are GET and OPTIONS only |
| Authorization | All backend source inspected | No authentication middleware, sessions, roles, ownership checks, or account suspension |
| Tests | Backend test files | Config, pool config, health handler/service; no marketplace round trips |
| Frontend | `frontend/src/App.tsx` | Root and unknown paths redirect to component showcase |
| HTTP adapter | `frontend/src/api/http/HttpJobsRepository.ts` | Calls proposed `/jobs` and `/jobs/:id/saved` routes that do not exist in Go |
| Mock adapter | `frontend/src/api/mock/MockJobsRepository.ts` | Static examples; saves are memory-only and reset on reload |

Existing health response: `{"status":"ok","database":"up","service":"jobhub-ai"}`.
Existing error shape: `{"error":{"code":"NOT_FOUND","message":"Endpoint not found"}}`.
There are no existing marketplace request DTOs or response DTOs to match.

## Required capability inventory

Every capability below is absent from the current Go source and migrations.

| Area | Missing backend support | Priority |
| --- | --- | --- |
| Authentication | Register, login, logout, current user, sessions, roles, suspension | MVP 1 |
| Jobs | List/detail/create/update/delete, publication lifecycle, ownership, filters | MVP 1 |
| Companies | List/detail/create/update, membership, public company jobs | MVP 1 |
| Applications | Submit, candidate list, employer applicants, status change, withdrawal | MVP 1 |
| Saved jobs | Candidate list/save/unsave | MVP 1 |
| Profile | Private candidate profile, structured experience/education/skills/preferences | MVP 1 |
| Resume | Upload/replace/delete/preview, private storage and authorized access | MVP 1 |
| Moderation | Admin provisioning, hide/remove/restore, reasons/history, suspension | MVP 1 |
| Reviews/ratings | Public list, submit/edit own review, moderation and server aggregation | MVP 2 |
| Following/searches | Follow/unfollow companies, persistent saved searches, alert preference | MVP 2 |
| Insights/retention | Salary/interview data, recent views, notifications | MVP 3 |
| Reports/verification | Report submission, queues, richer review tools, verification workflow | MVP 4 |

## Gaps in the frontend foundation

- Selecting the HTTP adapter does not make jobs integrated: every job endpoint currently returns 404.
- The HTTP adapter passes domain query property names directly to Axios. They differ from the intended URL query names (`query` vs `q`, arrays vs repeated singular keys). Explicit serialization is required.
- URL parsing currently casts untrusted strings to enums and accepts fractional page values. Shared validated schemas must replace these casts.
- The mock search accepts sort/date filters but does not implement them. It cannot verify real search behavior.
- `VITE_USE_MOCKS` currently defaults to true in every build, and the repository module imports both adapters. Production must reject mock mode and must not silently fall back to it after an HTTP failure.
- The current DTOs are frontend proposals, not a backend contract. `JobSummary` is insufficient for vacancy detail and edit forms.
- There is no session transport, CSRF handling, guest return-context flow, or server-driven role navigation yet.
- Showcase interactions do not prove persistence, authorization, application submission, or moderation. No product feature is considered complete based on this showcase.

## Scope and next step

The new integration brief supersedes a frontend-only definition of completion.
Keep React/Vite, the repository boundary, TanStack Query, Gin, pgx, sqlc, and
PostgreSQL/Supabase hosting. Implement MVP 1 in vertical slices against the real
Go API before starting company reputation or retention features.

See [the proposed MVP 1 contract](mvp1-api-contract.md). It explicitly distinguishes
new decisions from existing behavior. Backend implementation follows approval
because the earlier instruction kept backend changes out of scope and the new
brief requires missing contracts to be reviewed before backend changes.

## Verification performed

- Inspected all tracked Go source, routes, middleware, migration schema, SQL,
  generated model declarations, and tests relevant to this inventory.
- `cd backend && go test ./...` passed.
- `cd backend && go vet ./...` passed.
- No live database schema was introspected and no migration was executed.
  Conclusions describe repository-managed capabilities, not unversioned objects
  that may exist in a separately configured Supabase project.
- No runtime source, backend configuration, credentials, or existing Stage 1
  changes were modified by this audit.
