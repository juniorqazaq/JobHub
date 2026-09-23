# MVP 0: native and external vacancy schema proposal

Status: design only, pending approval. No SQL migration or integration added.
Read alongside [provider research](mvp0-vacancy-supply-research.md).
The existing [MVP 1 contract](mvp1-api-contract.md) is unchanged and remains the
next-phase native marketplace proposal.

## Core model

Use one future `jobhub.jobs` table with local UUIDs for stable JobHub URLs, and
explicit source identity. Do not use external IDs as local primary keys. Native
jobs retain their future company ownership/publication behavior. Imported jobs
do not require invented employer users, memberships or applications.

| Field/concept | Proposed meaning |
| --- | --- |
| `id` | Local UUID, stable across source updates |
| `source` | Non-null identity namespace: `jobhub`, `hh`, `jooble:kz`, `adzuna:gb` |
| `external_id` | Text; null only for native jobs; preserve provider identifiers exactly |
| `source_url` | Imported public landing/tracking URL; never a credential-bearing API URL |
| `upstream_source_name` | Original publisher label from an aggregator, distinct from ingestion source |
| `company_id` | Nullable FK to local companies for imports; native requirements stay in MVP 1 |
| `company_source_id` | Optional FK to a known provider employer identity |
| `company_name_raw` | Source-supplied company label even when no identity can be resolved |
| `application_method` | `external` for imported MVP 0 jobs; native internal behavior deferred |
| `apply_url` | Optional provider-authorized application URL; otherwise use source landing URL |
| `description`, `description_kind` | Plain/sanitized text and `full` or `snippet`; never imply a snippet is full content |
| `source_status` | External state: `unknown`, `active`, `archived`, `expired`, `removed` |
| `publication_status` | Future native lifecycle, independent of external status |
| `first_seen_at`, `last_seen_at` | Local first/latest observation in a successful provider response |
| `last_synced_at` | Last successful validation and normalization into the database |
| `fresh_until` | JobHub display freshness deadline, under provider-approved policy |
| `external_created_at`, `external_published_at` | Provider creation/publication times, nullable |
| `external_updated_at`, `external_expires_at`, `external_archived_at` | Provider events, nullable, never fabricated from local sync time |
| `created_at`, `updated_at` | Local persistence timestamps; separate from upstream history |

Provider family/market belong in a small `job_sources` registry, keyed by `source`.
Regional namespaces avoid assuming IDs from country-specific APIs are globally
unique. HH uses one namespace across its shared API. Never change namespace when
rotating credentials or when an aggregator's original publisher label changes.

The registry can describe enabled state, approved terms/version, attribution
template, retention/freshness policy and allowed display fields. Credentials
belong in server configuration or secret storage, never browser-visible rows.
Raw JSON storage is optional and retention-limited, subject to explicit rights;
do not create an unrestricted permanent mirror by default.

## Required identity constraints

Illustrative fragment for a future migration, not runnable DDL for the current
infrastructure-only schema:

```sql
UNIQUE (source, external_id),
CHECK (
  (source = 'jobhub' AND external_id IS NULL)
  OR
  (source <> 'jobhub'
    AND external_id IS NOT NULL AND length(btrim(external_id)) > 0
    AND source_url IS NOT NULL AND length(btrim(source_url)) > 0
    AND application_method IS NOT NULL AND application_method = 'external')
)
```

`source` is separately `NOT NULL` and a foreign key. PostgreSQL's ordinary UNIQUE
permits multiple native rows with null external IDs, while rejecting duplicate
imported `(source, external_id)` pairs. Do not use `NULLS NOT DISTINCT` here.

Ingestion will use an atomic `INSERT ... ON CONFLICT (source, external_id)
DO UPDATE`, retaining the local UUID and first-seen time. Concurrent runs must
not allow older observations to overwrite newer state. Serialize per source or
use an explicit observation/version guard. Missing optional fields in partial
responses must not erase previously retrieved detail fields.

This prevents duplicates within a provider namespace. It intentionally does
not merge the same real vacancy syndicated by multiple providers; future
cross-source duplicate groups must preserve provenance, rights and independent
removal behavior. Title/company string matching alone is not an identity rule.

## Companies

Keep `companies` as local entities. Add `company_sources` with local `id`,
`company_id`, `source`, `external_id`, source name/profile URL and sync timestamps;
enforce `UNIQUE(source, external_id)`. This supplies the requested
`external_company_id` concept without making one company's HH identity overwrite
its identity at another provider.

If a job carries `company_source_id`, enforce matching job source and local
company identity through composite keys/FKs (or equivalent transactional
validation). An import may have only `company_name_raw` and null company FKs.
Do not invent company IDs or automatically merge identical names. An imported
company is not automatically a verified JobHub employer. Future ownership
claiming requires a separate verified workflow; provider verification and
JobHub `is_verified` are distinct.

## Content and salary normalization

Preserve raw location and provider area IDs; normalize country/city only when
unambiguous. Preserve source language; interface i18n does not translate job
content. Multi-valued work formats must not collapse silently into a misleading
single value. Unknown experience does not mean junior; a years-of-experience
range is not a reliable seniority label.

Salary needs nullable decimal min/max, currency, period, gross/net (nullable
boolean), `salary_raw`, and `salary_is_estimated`. Unknown salary is not zero.
Do not infer monthly/KZT from the Kazakhstan search context. Parse free text
only with explicit, tested rules; retain unknowns when ambiguous. Filters and
sorting use comparable known currency/period values, matching the intent of the
MVP 1 proposal. Estimated values remain visibly distinct.

Normalize timestamps to UTC only with an offset or documented provider timezone.
Retain an offsetless raw timestamp (if permitted) and leave its normalized value
null until resolved. Do not substitute external update time for publication time.
Use an explicitly labeled first-observed date when publication is unavailable;
do not let periodic sync make an old vacancy appear newly published.

## Normalized Jooble POC mapping

This mapping follows the official [response specification](https://help.jooble.org/en/support/solutions/articles/60001448238).
The left column contains provider fields; the right column is our proposed design.

| Jooble field | JobHub destination |
| --- | --- |
| Adapter configuration | `source = 'jooble:kz'`; local UUID generated once |
| `id` | `external_id`, losslessly converted to text |
| `title` | `title` |
| `link` | `source_url`; outbound destination, preserved |
| `source` | `upstream_source_name`, never ingestion `source` |
| `company` | `company_name_raw`; company identity unresolved |
| `location` | `location_raw`; normalize only if unambiguous |
| `snippet` | `description`; `description_kind = 'snippet'` |
| `type` | Raw employment label; dictionary mapping or unknown |
| `salary` | `salary_raw`; structured amounts only when unambiguous |
| `updated` | `external_updated_at`, subject to timezone validation |
| No supplied equivalent | External publication/expiry/company ID stay null |
| Successful local ingestion | Set observation/sync timestamps and policy freshness |

Treat a returned search listing as recently observed, not proof of an explicit
provider `active` flag. Preserve `source_status = 'unknown'` where no status was
supplied. Proposed CTA is “View on source”; its final wording/attribution must
match the approved provider agreement. No internal application is created.

For a future approved HH adapter, map its employer ID through `company_sources`;
for Adzuna, retain its predicted-salary flag and outbound redirect URL. Neither
adapter is part of this phase.

## Freshness, removal and indexes

Track future `ingestion_runs` separately: source, requested scope, start/end,
completion status, cursor/pages, request budget and sanitized errors. A failed
run must not update job success timestamps or empty the database.

An item missing from a paginated, ranked, capped or filtered search is not proof
it expired. Explicit archive/expiry/removal signals can suppress it immediately.
Without those signals, use a provider-approved local freshness deadline to hide
unrefreshed listings; label the reason stale, not externally expired. The chosen
deadline is a JobHub policy, not an undocumented vendor guarantee. On permission
revocation, disable display and purge content according to the agreement;
even retained tombstones must follow allowed retention.

Suggested indexes: the required source/external-ID unique index; source plus
last-seen for reconciliation; company FKs for joins; eligible listing date plus
UUID for stable ordering. Add search indexes only against measured queries.
Do not put volatile `now()` freshness expressions in partial-index predicates;
evaluate freshness in reads and use static-state predicates where appropriate.

## Relationship to MVP 1

The preserved MVP 1 contract requires an active employer account/company for
native public vacancies. Imported jobs need a separately approved visibility
branch: authorized/enabled source, permitted fields, valid outbound URL, fresh
observation, and no known archive/removal/expiry. Do not fake employer accounts
to satisfy the native predicate. Unknown provider state can be displayed only
within the agreed freshness policy, with no guarantee that the source still
accepts applications.

Before implementation, formalize the imported public response variant: source
metadata, nullable company/salary fields, snippet distinction and external CTA.
The existing native write contract remains the next phase; this proposal does
not silently change it. Source fields will be server-managed and unavailable
to native employer edits. Public reads, details and counts must share their
respective visibility rules.

All future tables stay in `jobhub`, with RLS and deliberate server-role grants;
no direct browser/Supabase Data API access. Keep React/Vite → Go → PostgreSQL.
Auth, resumes, applications, employer ownership and admin moderation remain
deferred; no such systems are introduced by this design.
