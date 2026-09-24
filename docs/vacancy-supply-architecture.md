# Vacancy supply architecture

Status: architecture review and design only, 24 September 2026. No schema, API,
provider or runtime behavior changes are authorized by this document.

## Current JobHub ingestion architecture

The implemented flow is:

```text
Jooble REST search
        ↓
Jooble-specific client and vacancy DTO
        ↓
Jooble normalization
        ↓
In-run map dedupe by external ID
        ↓
Atomic PostgreSQL upsert
        ↓
Shared public visibility predicate
        ↓
GET /api/v1/jobs and GET /api/v1/jobs/:id
```

### What is already present

| Concern | Current behavior | Assessment |
| --- | --- | --- |
| Source registry | `jobhub.job_sources` stores source/provider/market/display name, enabled state, production permission and attribution text. | Good foundation. |
| Provenance | Jobs have `source`, `external_id`, `source_url`, `upstream_source_name`, raw company name and provider timestamps/status fields. | Sufficient for additional sources without provider-specific columns. |
| Identity | `UNIQUE (source, external_id)` and source identity checks distinguish native and imported rows. | Correct provider-level rule; retain it. |
| Company identity | Optional `company_sources` supports `(source, external_id)` without requiring a fake employer or automatic name merge. | Correct, currently unused by Jooble because its result has no stable employer ID. |
| Native/imported behavior | Native `source=jobhub` jobs require owned companies and internal application. Imports require source URL and `application_method=external`. | Correct separation. |
| Normalization | Jooble maps ID, link, upstream source, raw company/location/type/salary, snippet and updated time. | Honest mapping; missing provider data remains missing. |
| In-run dedupe | A map keyed by Jooble external ID removes repeat search results in one run. | Correct for one source. |
| Persistence | `INSERT ... ON CONFLICT (source, external_id) DO UPDATE` retains local UUID and `first_seen_at`; an observation-time guard prevents older runs overwriting newer data. | Correct idempotent base. |
| Atomicity | Source advisory lock, all row upserts, counters and successful run completion share one transaction. | A failed write rolls back job freshness and success state. |
| Failed request | Provider failure marks the run failed before any persistence step. Tests assert prior title/last-seen/last-synced values remain unchanged. | Meets the no-false-freshness requirement. |
| Run metadata | Status, searches, requests, fetched, normalized, inserted, updated, skipped, sanitized error code/message and timestamps are recorded. DB constrains request count to 0–50. | Good POC audit trail. |
| Freshness | Successful imports set first/last seen, last synced and `fresh_until`; Jooble defaults to a bounded freshness duration. | Works for POC, but policy is service-wide/provider-specific code rather than registry-driven. |
| Public visibility | Shared `jobhub.job_is_public` checks native lifecycle separately from imported `source_status` and `fresh_until`. Reads also require enabled source and, in production mode, confirmed permissions. | Strong safety boundary. |
| Public API | List/count/detail use the same source/permission/public predicates and return imported source metadata plus an external CTA. | Consistent. |
| Jooble secret safety | Backend-only key; client logs request count/search values rather than key-bearing URL. Request budget is enforced. | Retain. |

The schema already includes nullable `external_created_at`,
`external_published_at`, `external_updated_at`, `external_updated_raw`,
`external_expires_at` and `external_archived_at`. Future adapters should populate
only values explicitly supplied by the provider. No new column per provider is
needed for these concepts.

## Gaps before provider number two

1. **The service interface is Jooble-specific.** `JoobleSearcher`, `RunJooble`
   and direct calls to `jooble.Normalize` couple orchestration to one provider.
2. **Freshness is not source-policy driven.** A duration is injected into the
   Jooble service. The source registry does not yet express refresh interval,
   stale threshold, snapshot completeness, removal policy or terms version.
3. **No reconciliation phase exists.** Jobs absent from a successful complete
   snapshot are not counted or transitioned. This is correct for ranked Jooble
   searches, but a future full-feed adapter needs explicit reconciliation.
4. **External lifecycle fields are available but not exercised.** Jooble supplies
   only `updated`; imported rows remain `source_status=unknown` until freshness
   hides them.
5. **Run scope is coarse.** Counts exist, but run records do not retain a safe
   scope identifier, page/cursor progress, snapshot completeness, provider
   response watermark or malformed count.
6. **No cross-provider duplicate diagnostics exist.** Provider-level identity is
   safe, but syndicated copies can appear as separate public rows.
7. **Permission metadata is minimal.** The boolean gate is effective, but future
   governance needs approved fields, terms/agreement version, approval date,
   attribution policy and deletion SLA. This can initially live in configuration
   or internal documentation; a migration should be evidence-driven.
8. **Supply health has logs and run rows but no internal roll-up.** Operators
   cannot yet see stale/hidden/duplicate distributions by source at a glance.

These are design gaps, not reasons to rewrite the working Jooble POC now.

## Provider-neutral target model

Keep provider packages responsible for transport and source semantics; keep the
orchestrator responsible for bounded execution and atomic persistence.

```text
Provider adapter
  Fetch(scope, cursor, budget) → provider page + request usage
        ↓
Provider vacancy DTO
        ↓
Normalize → canonical ImportedJob + validation issues
        ↓
Validate required identity, URL, content and provider policy
        ↓
Provider-level dedupe by (source, external_id)
        ↓
Atomic upsert + optional complete-snapshot reconciliation
        ↓
Source-aware freshness/lifecycle policy
        ↓
Shared public visibility
        ↓
Public jobs API
```

A future interface can conceptually provide:

- stable `Source()` and declared capabilities;
- bounded `Fetch` returning items, cursor/page metadata and exact request use;
- `Normalize` returning a canonical job plus structured skip reason;
- snapshot type: ranked/partial, complete active snapshot, or event/delta;
- provider watermark and lifecycle signals where documented.

The canonical model should remain source-neutral. Keep raw provider labels and
optional structured values, but do not add `jooble_*`, `enbek_*` or `hh_*`
columns. Provider-only diagnostic payloads, if contractually allowed, belong in
a retention-limited ingestion artifact store, not the public jobs table.

### Validation boundary

Before persistence, every imported job must have:

- registered and enabled source;
- nonblank external ID and title;
- safe public `https` source/apply URL with no credentials;
- external application method;
- a known observation time;
- description kind that truthfully distinguishes snippet from full text;
- no invented salary, publication time, company identity or lifecycle state.

Provider permission gating remains independent of technical validity. A valid
row from an unapproved source stays hidden in production.

## Source-aware freshness strategy

Native JobHub jobs retain their existing publication, expiry, moderation,
ownership and company/user-state lifecycle. Imported freshness rules must never
expire native jobs.

Each imported source needs an approved policy with:

| Policy input | Purpose |
| --- | --- |
| Expected refresh interval | Detect scheduler/provider lag. |
| Maximum safe display age | Hide a job when no reliable lifecycle signal exists. |
| Feed completeness | Decide whether absence has meaning. |
| Archive/expiry/delete fields | Apply immediate provider-declared closure. |
| Missing-item threshold | Require one or more complete successful snapshots before action. |
| Removal action | Hide, tombstone or purge as the agreement requires. |
| Attribution/field permissions | Control what the API may expose. |
| Production permission state | Preserve the existing final visibility gate. |

### Lifecycle precedence

1. A documented provider `removed`, `archived`, `expired` or `closed` signal
   hides the job immediately and records the corresponding source status/time.
2. A provider expiry timestamp hides the job at expiry unless a later provider
   update explicitly extends it.
3. Absence from a **complete, successfully fetched snapshot** may transition a
   job only under that source's approved missing-item rule.
4. Absence from ranked search, capped pagination, a filtered query or a failed or
   incomplete run is never proof of closure and must not refresh or remove it.
5. When lifecycle signals are unavailable, `fresh_until` is computed from the
   source-specific maximum safe display age. Passing it hides the row as stale;
   it does not claim the provider marked it expired.

Illustrative policies, pending provider agreement:

- **Jooble POC:** ranked/partial searches; no absence reconciliation; short local
  stale deadline; `source_status=unknown`; development only.
- **Enbek:** if a complete active snapshot is documented, refresh from that
  snapshot and reconcile only after full success. Respect explicit archive or
  expiry first. Do not assume the public one-month rule equals feed semantics.
- **Employer ATS board:** authorized complete board snapshot; a missing posting
  can become a duplicate/removal candidate, with a short confirmation window or
  second successful snapshot unless the ATS documents immediate removal.

Failed runs never move `last_seen_at`, `last_synced_at`, `fresh_until` or source
status. A successful partial run refreshes only returned items.

## Cross-provider duplicate candidate detection

Keep `UNIQUE(source, external_id)` as the only automatic identity merge. Do not
merge companies or jobs solely by name, and do not delete suspected duplicates.

Add a future diagnostic layer that emits duplicate candidates with a score,
reasons, timestamps and review state. Candidate signals can include:

- exact normalized/canonical source URL;
- employer-authorized canonical vacancy URL;
- verified company-source link, never raw name alone;
- normalized title and location;
- publication times within a source-aware window;
- salary currency/period/range compatibility;
- description fingerprint or high token similarity;
- matching employment/work mode and requisition reference.

Suggested confidence logic:

- **Very high:** same canonical employer vacancy URL, or an explicit upstream ID
  bridge supplied by a provider.
- **High:** verified same company plus strong title/location/description match and
  compatible publication window.
- **Medium:** raw company/title/location match with supporting salary/content
  similarity; diagnostic only.
- **Low:** title and city alone; collect metrics but do not affect display.

Store pair/group identifiers and individual reason codes. Later, a deterministic
display policy can prefer an employer-authorized native job, then a direct
employer feed, then an aggregator, while preserving every source's provenance
and independent removal obligations. That policy should be based on observed KZ
duplicate patterns, not designed from synthetic examples.

## Supply observability

Build an internal, server-only source health view before scaling imports. It can
start as SQL/internal logs rather than a public dashboard.

### Per-source summary

| Metric | Meaning |
| --- | --- |
| Active | Publicly eligible imported jobs now. |
| Fetched / normalized | Last run transport and usable totals. |
| Inserted / updated / skipped | Existing ingestion counters. |
| Stale | Passed local freshness deadline. |
| Hidden | Disabled source, unconfirmed permission, source lifecycle or validation. |
| Errors | Failed runs and sanitized error category. |
| Last successful sync | Source-level freshness heartbeat. |
| Requests used | Last run, rolling day/month and provider quota window. |
| Duplicate candidates | Counts by confidence and source pair. |
| Malformed | Counts by validation reason. |
| Missing salary/company | Data-quality percentages. |
| Freshness distribution | Age buckets based on last seen and provider update. |

Alert on consecutive failures, scheduler lag beyond expected refresh, sudden
count collapse/growth, quota exhaustion, malformed-rate change, stale percentage,
permission expiry and removal backlog. Never include secrets, key-bearing URLs,
raw private data or full provider payloads in logs.

## Scaling path

### 100 jobs

- Keep Jooble as a bounded development POC only.
- Validate freshness and duplicate diagnostics with small samples.
- Onboard several employer-authorized ATS feeds if available.
- Establish source runbooks, contacts and written permission records.

### 500–1,000 jobs

- Add Enbek only after the stop/go criteria in the strategy document pass.
- Run bounded complete-snapshot ingestion with provider-specific reconciliation.
- Add internal source health and quota tracking before increasing cadence.
- Measure cross-source duplicates and data completeness; do not chase raw count.

### 10,000 jobs

- Require sustainable quotas/contract SLAs and incremental or partitioned sync.
- Schedule sources independently with jitter, backoff and per-source locks.
- Separate fetch artifacts from canonical persistence with strict retention.
- Add repair/replay tooling that cannot refresh jobs from failed or partial runs.
- Review indexes and search capacity from measured query plans.
- Add formal removal SLA monitoring and auditable provider agreement versions.

At every scale, production visibility remains conditional on source approval,
successful freshness, valid provenance and external application behavior.

## Import safety checklist for every future provider

- bounded request budget enforced in code and recorded in the run;
- server-only secrets and no API key or credential-bearing URL in logs;
- source registry entry defaults to production permission false;
- canonical validation before persistence;
- provider-level dedupe plus atomic guarded upsert;
- failed/partial runs do not refresh missing or existing jobs incorrectly;
- attribution and allowed fields derived from the written agreement;
- explicit external CTA; no fake employer accounts or native applications;
- lifecycle reconciliation enabled only for documented complete feeds;
- immediate kill switch through source enabled/permission state;
- removal/purge runbook tested before production approval.
