# Greenhouse development POC

Implemented 25 September 2026, following [ATS source research](ats-vacancy-sources.md).
Only Greenhouse is added. There is no scheduler, production enablement, new schema,
remote migration, Lever implementation or frontend change.

## Contract and mapping

The adapter performs one GET to the fixed official host:
`https://boards-api.greenhouse.io/v1/boards/{board_token}/jobs?content=true`.
It does not request HTML pages, application questions or credentials. GET needs no
API key. Redirects are refused; errors never include request URLs, response bodies
or wrapped transport errors. No retries or detail enrichment are performed.
See the [official Job Board contract](https://docs.greenhouse.io/job-board.html).

| Greenhouse field | Existing provider-neutral `jobs.ImportedJob` / persistence |
| --- | --- |
| Posting `id` | Lossless positive decimal string `external_id`; never float64, requisition ID or internal job ID |
| Configured board token | Source key `greenhouse:<board_token>`; no real tokens compiled into code |
| `title` | Plain-text title, required |
| `absolute_url` | Exact HTTPS URL retained in `source_url` and `apply_url`, including tracking query; credentials rejected |
| `content` | Entity-decoded plain text; script/style content discarded; description kind `full` |
| `location.name` | Raw location text; no guessed city/country or remote eligibility |
| `company_name`, if supplied | Raw company name; no automatic company ownership/link |
| `updated_at`, if supplied | Parsed UTC timestamp and original timestamp; invalid value rejects the record |
| `internal_job_id: null` | Prospect post skipped, not treated as an open vacancy |
| Missing salary/type/work mode/company/publication fields | Remain empty in the Go import model and NULL in nullable DB columns; sample values are nullable |

No inference from department names, free-text salaries or arbitrary metadata.
No additional provider-specific vacancy columns. Native application paths remain
unchanged; persisted ATS jobs always use external application behavior.

## Shared execution and identity

`providers.VacancyProvider.Collect` returns normalized records, request/fetched/
malformed/skipped counts and a completeness observation. `ingestion.Service.Run`
owns canonical validation, deterministic ID-sorted samples, exact-ID dedupe, run
records and the existing atomic PostgreSQL upsert. `RunJooble` now wraps the same
path and keeps its configured freshness duration and partial-search semantics.

The existing `UNIQUE(source, external_id)` preserves the local UUID and first-seen
time. Collection start is the observation timestamp, so the existing SQL guard
rejects an older run overwriting a newer observation. Duplicate records count as
skipped; the last valid occurrence wins, and duplicates invalidate completeness.
Failures never persist collected vacancies. Failure messages are safe categories.

Completeness requires a valid jobs array, matching `meta.total`, no malformed
records and no duplicate IDs. Missing totals, caps, malformed data and fetch errors
cannot claim a complete snapshot. A recognized prospect is intentionally excluded.
This flag is evidence about retrieval, not an atomic upstream snapshot guarantee.

**There is no disappearance reconciliation or automatic ATS expiry in this POC**,
even for complete snapshots. The development policy explicitly supplies a year-9999
`fresh_until` because the existing column is mandatory. This is a local display
sentinel, not a provider expiry claim or production freshness policy. Empty boards
do not expire or refresh previous rows. Disable/delete development sources after
the experiment; production lifecycle policy remains a separate implementation.

## Server-side configuration and bounds

Use a private JSON configuration file outside the repository for real boards:

```json
{
  "boards": [
    {
      "board_token": "REPLACE_WITH_AUTHORIZED_BOARD",
      "display_name": "Authorized employer display label",
      "authorized_for_poc": false,
      "test_board": false
    }
  ],
  "max_requests": 12,
  "max_jobs": 200
}
```

Set `authorized_for_poc=true` only after recording employer/platform permission
for this development experiment. It is an operator attestation, not automatic
ownership verification, and never changes production permission. The display label
is source metadata, not an asserted company identity.

Configuration permits at most two ordinary boards plus one optional test board,
12 requests and 200 processed raw records. Duplicate and malformed records consume
the job budget. Budgets are shared across selected boards **within an invocation**;
track all live invocations against the experiment's cumulative 12-request limit.
Do not repeatedly restart the command to expand the authorized allowance.

Responses are limited to 5 MiB, with a 20-second HTTP timeout, a two-minute command
deadline and at least one second between selected boards. Greenhouse lists cannot
be paginated by this adapter: oversized boards are rejected before normalization,
never silently truncated into a supposedly complete snapshot. The server may
transmit more than 200 entries before the client can count them. Select genuinely
small boards in advance. No endpoint override or arbitrary-host discovery exists.

## Dry-run

From `backend/`, run the included synthetic example without any network or DB:

```sh
go run ./cmd/jobhub collect \
  --config internal/providers/greenhouse/testdata/config.json \
  --dry-run \
  --fixture internal/providers/greenhouse/testdata/board.json
```

For an authorized live board, omit `--fixture`, select the source from private
config, and retain `--dry-run`:

```sh
go run ./cmd/jobhub collect --config /private/path/ats.json \
  --source greenhouse:REPLACE_WITH_AUTHORIZED_BOARD --dry-run --sample 5
```

Exactly one of `--dry-run`, `--ingest` or `--health` is required. No implicit write
mode exists. `--fixture` is dry-run-only and requires one selected source. Staging
and production environments are refused, including dry-run.

Dry-run does not load a database URL, connect to PostgreSQL, register a source,
create ingestion runs, mutate health, or advance any cursor. It returns request,
fetched, normalized, malformed, skipped and duplicate counts, completeness and up
to ten samples. Samples omit descriptions/raw payloads and remove URL query/fragment
values; persistence keeps the original URL. In fixture mode, `RequestCount=1`
means one simulated adapter request and `network_requests=0` states the actual
network use. Inserted/updated values are zero, not predicted persistence counts.

A valid response with malformed records succeeds as a partial collection and
reports those skips explicitly; fetch/schema/size/budget failures return a nonzero
exit status. Review `complete_snapshot` and counters, not just process exit status.

## Optional isolated ingestion and source health

Ingestion requires explicit `APP_ENV=development`, a separate
`ATS_DEV_DATABASE_URL`, a numeric loopback host, explicit port and a database name
beginning `jobhub_ats_poc_`. Only the `sslmode` connection query option is accepted;
remote host/service overrides are rejected. `DATABASE_URL` is never used by this
command. Set credentials through the server environment without printing them.

Prepare an isolated local database using the existing migrations; the command
never runs migrations. Then use the same private authorized configuration:

```sh
go run ./cmd/jobhub collect --config /private/path/ats.json \
  --source greenhouse:REPLACE_WITH_AUTHORIZED_BOARD --ingest

go run ./cmd/jobhub collect --config /private/path/ats.json \
  --source greenhouse:REPLACE_WITH_AUTHORIZED_BOARD --health
```

Only isolated ingestion registers sources, with production permissions false.
An existing disabled, incompatible-provider or production-approved source is
refused rather than overwritten/enabled. The normal public pipeline can display
local development results, while production-mode list/count/detail exclude them.

Health is a read-only projection of `job_sources` and `ingestion_runs`:

- `last_attempt`: newest run's persisted start time;
- `last_success`: latest successful run completion, unchanged by failed runs;
- `request_count`, `jobs_seen`: latest run's actual counters;
- `failure_category`: latest run's sanitized error code, NULL on success;
- execution status: never synced, running, failed, disabled, or successful
  development-only; a production-approved source is flagged for review.

A successful empty board has zero jobs and is not failing. This small projection
is execution health, not a claim of complete coverage or production approval.
Completeness/malformed diagnostics are in the collection report; no new persisted
metrics or cached registry fields are invented. Dry-run health is not persisted.

## Validation performed

No live employer board was supplied/authorized during this implementation turn.
**External Greenhouse requests: 0. Live jobs fetched: 0.** The following results
are from synthetic fixtures and an isolated local PostgreSQL instance.

| Check | Result |
| --- | --- |
| Offline CLI dry-run | 3 fetched, 2 normalized, 1 malformed/skipped, 0 duplicates; complete=false; 1 simulated request, 0 network requests, 0 DB mutations |
| HTTP-to-PostgreSQL fixture | First fetch inserted 1; second fetch updated 1 and inserted 0; one persisted row, same UUID and first-seen timestamp |
| Exact identity / URL | Large ID beyond float64 integer precision retained exactly; original URL including tracking retained; external CTA verified |
| Unknown fields | Salary, employment, workplace, company link and upstream publication time remain NULL |
| Failed fetch | HTTP 500 recorded as failure; title/last-seen/last-synced unchanged; last successful run retained |
| Empty snapshot | Successful zero-job run; prior job neither expired nor refreshed |
| Internal applications | Existing candidate store rejects ATS application; no application row created |
| Dry-run write guard | Panic-on-write store and real PostgreSQL read-only connection; no new ingestion run or vacancy change |
| Secret hygiene | Transport/body/argument sentinel secrets absent from errors/logs; query values removed from samples |
| Limits | Oversized responses, malformed containers/IDs/URLs/times, shared concurrent request budget, source-count caps, production/remote DB refusal tested |
| Required checks | `go test -race ./...` with isolated PostgreSQL, `go vet ./...`, `go build ./...`, `git diff --check` |

The HTTP persistence scenario used five loopback fixture requests, not provider
quota: first sync, second sync, dry-run, failed fetch, empty board. Tests clean
up their source/job/run rows. No new migration or sqlc query/generated-code change
was needed; all nine existing migrations were applied only to the disposable
local test database.

## Remaining gates

Before live collection: identify up to two authorized small boards, confirm size,
record permission, and keep the cumulative live request budget at twelve. Before
production: resolve storage, full-text redistribution, commercial use, attribution,
removal and retention rights; implement/review freshness and source ownership;
replace this development-only registry/configuration boundary with an approved
production onboarding design. Public GET availability and a passing POC do not
satisfy these gates. No production source is enabled by this change.
