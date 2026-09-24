# ATS vacancy sources and Source Registry proposal

Research: **24–25 September 2026**. Research and architecture only; no adapter,
migration, database write, scheduled collection, frontend or POC is implemented.

## 1. Executive summary

Employer ATS feeds can make onboarding configuration-driven: implement one
adapter, then register independently authorized company boards. They cannot
provide a legally unrestricted, automatically discovered Kazakhstan catalogue.
Recommend **Greenhouse first**, **Lever second**, conditional on source permission.
Both have documented public retrieval, posting IDs and straightforward board
collection. Kazakhstan hiring examples exist for both. Workday has relevant
employers but no verified official anonymous CXS integration contract; defer it.
Ashby's opt-in partner feed is promising for a later distribution agreement.

Technical candidates: Greenhouse, Lever, Ashby, SmartRecruiters, Workable,
Recruitee, Teamtailor and Personio. Recruitee and Teamtailor need employer-managed
authentication; SmartRecruiters' current authentication wording needs clarification.
BambooHR's documented ATS access requires authorization and does not establish a
complete public vacancy feed. Workday needs a supported employer/partner channel.
None is approved for JobHub production by this research.

The existing [supply strategy](vacancy-supply-2026.md) and
[ingestion architecture](vacancy-supply-architecture.md) remain the broader context.
This document refines ATS access and configuration, not marketplace design.
Its A–D labels below describe **access/rights**, not the strategy's priority classes.
No additional ATS was needed to resolve the first two implementation choices.

### Evidence and limitations

Official developer/support/legal documentation is primary. Employer-controlled
ATS pages, found through web search, establish possible KZ relevance, not current
feed counts or authorization. Search indexing may retain closed jobs. No job API
was called, no collector or crawler was run, and no provider quota was tested.
Public GitHub source was read, not executed. Unknown means the reviewed evidence
does not establish the property; it does not mean the provider lacks it.

## 2. ATS comparison

These labels are independent, not a ladder:

- **A:** anonymous technical retrieval documented; not a redistribution licence.
- **B:** documented use for the employer's own career site.
- **C:** third-party distribution explicitly supported through an approved,
  employer-opted-in partner channel; not automatic permission for JobHub.
- **D:** JobHub's proposed storage/display/commercial rights remain unconfirmed.

All endpoints below use HTTPS. `{board}`, `{site}`, `{company}`, `{tenant}` and
`{id}` are opaque identifiers, never values derived from a company display name.
Tables are split by concern to keep all requested dimensions readable.

### Access, tenant identity and retrieval

| ATS / official reference | Public feed; auth; classes | Tenant identifier | Listing endpoint | Detail endpoint | Pagination |
| --- | --- | --- | --- | --- | --- |
| [Greenhouse](https://docs.greenhouse.io/job-board.html) | Public GET; no key. A/B/D | Board token | `boards-api.greenhouse.io/v1/boards/{board}/jobs?content=true` | Same path + `/{id}` | No paging documented; board list + `meta.total` |
| [Lever](https://github.com/lever/postings-api) | Public GET; no key. A/B/D | Site slug + global/EU region | `api.lever.co/v0/postings/{site}?mode=json` (EU: `api.eu.lever.co`) | Same path + `/{id}` | `skip`, `limit` |
| [Workday](https://community-content.workday.com/en-us/public/products/platform-and-product-extensions/soap-api-reference.html) | Authenticated enterprise integrations; anonymous career frontend is not a supported public API contract. D | Tenant + datacenter host + career-site name | Official Recruiting web services / configured employer export; no verified universal anonymous listing endpoint | Contract-specific | Contract-specific |
| [Ashby](https://developers.ashbyhq.com/docs/public-job-posting-api) | Public board GET, no key. A/B/D; separate C partner feed | Case-preserved board name | `api.ashbyhq.com/posting-api/job-board/{board}?includeCompensation=true` | No separate public detail documented; description in list | None documented |
| [SmartRecruiters](https://developers.smartrecruiters.com/docs/endpoints) | Public Posting API examples omit auth; newer guide discusses key/OAuth/internal postings. B/D; A conditional; separate C | Career-site company identifier | `api.smartrecruiters.com/v1/companies/{company}/postings` | Same path + `/{id}` | `offset`, `limit`, `totalFound` |
| [Workable](https://help.workable.com/hc/en-us/articles/115012771647-Using-the-Workable-API-to-create-a-careers-page) | Public account feed; SPI bearer `r_jobs`. A/B/D | Account subdomain | `www.workable.com/api/accounts/{tenant}?details=true`; SPI `{tenant}.workable.com/spi/v3/jobs?state=published` | Public separate detail unverified; SPI `/jobs/{shortcode}` | Public unspecified; [SPI](https://workable.readme.io/reference/jobs): `limit`, `since_id`, `max_id` |
| [Recruitee](https://docs.recruitee.com/reference/offers) | Careers token required by current support guidance; enforcement deadline below. B/D | Career-site subdomain | `{tenant}.recruitee.com/api/offers/` | `/api/offers/{id}` per [auth documentation](https://docs.recruitee.com/reference/authentication-1) | No paging documented on Careers list page; do not import ATS API pagination assumptions |
| [BambooHR](https://documentation.bamboohr.com/reference/get-job-summaries) | Authorized ATS summary API, not verified anonymous feed. D | Company subdomain | `{tenant}.bamboohr.com/api/v1/applicant_tracking/jobs` | Public full-job detail API unverified | No pagination contract established for summaries |
| [Teamtailor](https://docs.teamtailor.com/) | Token with public-read scope; API-version header. B/D; separate C | Credential-bound account + EU/NA stack; career slug for detection | `api.teamtailor.com/v1/jobs` (NA: `api.na.teamtailor.com`) | `/v1/jobs/{id}` | JSON:API `links.next`, `page[size]`; follow returned cursor links |
| [Personio](https://developer.personio.de/docs/retrieving-open-job-positions) | Employer-enabled XML, no credentials. A/B/D | Account + `.jobs.personio.de` or `.com` | `https://{tenant}.jobs.personio.de/xml` or `.com/xml` | No separate XML detail; full blocks in feed | None documented |

### Fields and original application destination

“Unverified” deliberately distinguishes undocumented schemas from absent values.
Optional fields remain nullable. Never promote application-question salary or
location fields into vacancy compensation/location.

| ATS | Stable external ID | Company / department | Location | Salary | Workplace / employment | Full description | Original application destination |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Greenhouse | Posting `id`, not `internal_job_id` | Detail `company_name`; departments | `location.name`, offices | Optional detail pay ranges | Custom metadata; no universal standard mapping | `content=true` | `absolute_url`, hosted or employer career page |
| Lever | Posting `id` | Configured company; categories team/department | Location/allLocations, country | Optional `salaryRange` | `workplaceType`; commitment | Description + lists + additional sections | `applyUrl`; preserve `hostedUrl` separately |
| Workday | Not established for approved feed; requisition and posting identity may differ | Feed-specific | Feed-specific | Feed-specific | Feed-specific | Feed-specific | Employer-approved posting URL; do not invent API from HTML |
| Ashby | Public schema does **not document a standalone ID**; URL identity requires confirmation. Partner feed explicitly has UUID | Configured company; department/team | Primary/secondary addresses | Optional compensation tiers | `workplaceType`, `isRemote`; `employmentType` | HTML/plain | `applyUrl`, `jobUrl` |
| SmartRecruiters | `id` or UUID; choose one consistently | Company identifier/name, department | Country/region/city/coordinates | Not established in reviewed public schema | `location.remote`; employment type | Detail `jobAd.sections` | `applyUrl` |
| Workable | SPI `id` and shortcode; public schema stability requires fixture verification | Account; SPI department | SPI structured/multiple locations | SPI salary bounds/currency | SPI workplace/telecommuting, employment data; public parity unverified | Public `details=true`; SPI detail | SPI `application_url`/`url`; public field contract to verify |
| Recruitee | Detail identifier supported; exact response ID type/stability needs schema fixture | Tenant; department filter verified | Multi-location examples documented | Unverified in accessible reference | Unverified in accessible reference | Exact field contract unverified | Hosted offer route exists; returned URL field must be verified |
| BambooHR | Summary job identity needs schema confirmation | Account; response fields unverified | Unverified | Unverified | Unverified | Not established by summary endpoint | No verified response URL field; obtain supported public channel |
| Teamtailor | JSON:API resource `id` | Account, department relation | Location relations | Min/max, currency, salary unit | Remote status, employment type | Job body | Career-site apply link; external-application URL if configured |
| Personio | Position `id` | Subcompany/department | Office text | Not in documented XML schema | Employment type + schedule; remote not documented | HTML description blocks | [Documented construction](https://developer.personio.de/docs/integration-of-open-positions): same board host + `/job/{id}`; verify original host |

### Time, completeness and lifecycle

A complete active board is not an atomic snapshot unless the provider guarantees
it. “Absence candidate” means no longer listed, not proof that a role was filled.

| ATS | Publication / update time | Complete listing? | Closure / archive signal | Proposed freshness behavior |
| --- | --- | --- | --- | --- |
| Greenhouse | `updated_at`; detail `first_published`, deadline | Whole board documented | No tombstones documented; deadline may exist | Reconcile missing IDs after two complete observations; respect deadline |
| Lever | Public publication/update fields not documented | All published postings after paging | Disappearance; no public tombstone stream | Two complete observations; local observation age, not fabricated publication |
| Workday | Unverified approved-feed contract | Unknown until feed agreement | Unknown | No absence reconciliation until supported feed semantics verified |
| Ashby | Public `publishedAt` is last publication, not general update; partner `updatedAt` | Public published collection; exclude `isListed=false` | Unlisting / absence; no public archive timestamp | Reconcile listed subset only, after identity and completeness validated |
| SmartRecruiters | `releasedDate`; republishing needed to update posting content | Active postings after all pages | Detail `active`; list disappearance | Explicit inactive hides; otherwise two complete observations |
| Workable | SPI `created_at`, `updated_at`; public parity unknown | Public published list; SPI must exhaust published scope | SPI draft/published/closed/archived | Never publish non-published SPI records; validate public snapshot before absence policy |
| Recruitee | Public timestamp fields unverified | Published collection documented; validate schema/completeness | Published-list absence, tombstones unverified | Enable absence only after token-backed full-collection validation |
| BambooHR | Unverified job timestamps | Summaries default to non-deleted, not public-only | Status filters include Open/Filled/Deleted/On Hold/Canceled | Status alone cannot establish public visibility; defer import |
| Teamtailor | Created/updated filters; date/publication semantics require version-specific validation | All published public pages | Status and feed filters; archived requires broader scope | Public-only enumeration; do not request internal data to learn closure |
| Personio | Example `createdAt`; no update timestamp documented | Current open XML positions | Disappearance; no archive flag documented | Complete XML + two observations; don't treat creation as publication |

## 3. Detailed provider findings and rate limits

### Greenhouse

Best first implementation: one list request can carry descriptions. Preserve
numeric IDs as decimal strings without float conversion. Distinguish prospect
posts from normal requisitions before inclusion. No numeric public GET quota was
found in the [Job Board reference](https://docs.greenhouse.io/job-board.html).
[Pay-transparency guidance](https://support.greenhouse.io/hc/en-us/articles/10028084062491-Add-pay-transparency-to-a-job-post)
confirms optional detail parameters; salary enrichment must fit the request budget.
The [MSA, section 4(d)](https://www.greenhouse.com/master-subscription-agreement)
makes order-form limits and fair use relevant even when a third party acts for a
customer. Do not borrow Harvest limits for Job Board GETs.

### Lever

Global and EU hosts must remain distinct. No numeric retrieval limit was found
in the [official public Postings API](https://github.com/lever/postings-api);
application POST limits are not GET limits. Assemble all documented description
sections. Its [authenticated Data API changelog](https://hire.lever.co/developer/updates)
now documents deleted postings, but that capability must not be attributed to the
anonymous Postings API. It does not justify adding credentials or applicant access.

### Workday

[Integration Cloud documentation](https://www.workday.com/content/dam/web/en-us/documents/datasheets/workday-integration-cloud-connectors-hcml-datasheet.pdf)
describes a Job Postings connector exporting active postings to external sites.
That is a direction for employer-authorized investigation, not a working endpoint
or verified current entitlement. Community collectors use CXS routes, but official
anonymous retrieval, stable identity, pagination, rates and lifecycle guarantees
were not established. Do not implement undocumented CXS as if it were public API.
[Site terms, API section](https://www.workday.com/en-us/legal/site-terms.html)
require authorization under a separate Workday agreement; establish applicable
tenant terms and supported channel first. No WAF/browser workaround is proposed.

### Ashby

Public `isListed=false` jobs must not enter public search. The public reference's
missing explicit job-ID contract is a reason to verify identity before coding,
not to invent a title hash. The [dedicated partner feed](https://developers.ashbyhq.com/docs/dedicated-partner-job-feeds)
provides posting/organization UUIDs, publication/update times and hourly JSON/XML
for opted-in employers. This is the strongest explicit aggregation direction in
this review, but partner provisioning, retention, rate and commercial terms still
need agreement. Public and partner schemas must remain separate capabilities.
No numeric public-board GET quota was established.

### SmartRecruiters

The [Posting API guide](https://developers.smartrecruiters.com/docs/posting-api)
now discusses API key/OAuth and internal postings, while endpoint examples show
anonymous requests. Ask whether external-only reads remain anonymous for the
selected tenant. Never broaden access to internal vacancies to resolve this.
[Customer throttling](https://developers.smartrecruiters.com/docs/throttling-policies)
is 10 requests/second and eight concurrent requests per API user; anonymous limit
scope needs confirmation. The [partner Job Board API](https://developers.smartrecruiters.com/docs/partners-job-board-api)
uses a partner key and purchased publication relationship. It is not a free
cross-company discovery endpoint.

### Workable

Prefer the documented public account feed over reverse-engineered career-page
routes. The public help page does not promise every SPI field. The
[SPI jobs reference](https://workable.readme.io/reference/jobs) supports update
filters and descriptions, but older non-published jobs may lack state-transition
history. A delta alone therefore cannot establish a complete initial snapshot.
No numeric public-feed quota was verified; employer SPI access is a separate
credential/permission choice.

### Recruitee

The [August 2026 support guidance](https://support.recruitee.com/en/articles/1066282-api-documentation)
says Careers API requires a Careers token. The
[authentication reference](https://docs.recruitee.com/reference/authentication-1)
sets **10 February 2027** as the unauthenticated-call cutoff, with header
`X-Careers-Sites-Token`. Plan authenticated onboarding now. XML/widget access is
explicitly outside that token's scope; it is not an excuse to bypass authorization.
The accessible list reference omits the rendered response schema, so salary,
timestamps, URLs and employment fields remain verification items. No numeric
Careers API limit established; do not substitute Company API limits.

### BambooHR

The [job summaries reference](https://documentation.bamboohr.com/reference/get-job-summaries)
requires ATS access and defaults to non-deleted jobs, including non-public states.
It is insufficient by itself for JobHub publication. Public `/careers/list`
patterns in collectors are not an official supported feed contract.
[Developer terms](https://www.bamboohr.com/legal/developer-terms-of-service),
sections 1.2, 3.2–3.3 and 5, require written customer authorization and approved
third-party authentication (OAuth or another approved mechanism), prohibit
customer-key handoff, constrain caching and require deletion/anonymization within
30 days of relevant termination/revocation, subject to legal exceptions. They do
not establish permission for an unrestricted commercial job mirror.
[The September 16, 2026 change](https://documentation.bamboohr.com/docs/planned-changes-to-the-api)
uses 429 with `Retry-After` for throttling; older 503 guidance is stale. A numeric
job-read allowance was not established.

### Teamtailor

“Public” describes token scope, not anonymous authentication. Use public read
only, a pinned tested API version, and returned pagination links.
[API limits](https://docs.teamtailor.com/) are 50 requests per ten seconds.
The [job-board partner channel](https://partner.teamtailor.com/job_boards/)
supports opted-in employer ads, update/removal webhooks and a current-state XML
feed generated three times daily; removed/unlisted ads must be removed downstream.
That channel has different cadence and lifecycle from polling a customer token.
No candidate, offer or internal-job permissions are needed.

### Personio

[The XML FAQ](https://support.personio.de/hc/en-us/articles/29375445597725-Frequently-asked-questions-on-XML-job-integration)
confirms employer activation, no credentials, multiple-site integration and hourly
sync advice. It does not establish unrestricted aggregation rights or a numeric
rate quota. Legal entities share a feed: filtering must not create duplicate
source identities or assign every subsidiary job to one company. Preserve feed
language independently of JobHub's UI locales; no English UI is added.

## 4. Terms and permission matrix

These are findings about reviewed evidence, not a legal clearance. “Unclear” is
a production blocker, not a claim that use is prohibited. Customer subscriptions
and permission to build a career page do not grant JobHub a data licence.

| ATS / rights source | Caching / storage | Public display / redistribution | Attribution | Commercial restrictions | Employer authorization |
| --- | --- | --- | --- | --- | --- |
| [Greenhouse MSA](https://www.greenhouse.com/master-subscription-agreement) | JobHub retention not granted by public GET docs | Customer-career use established; cross-company reuse unclear | No universal public-feed badge rule found | Order-form/API fair-use constraints; commercial aggregation scope unconfirmed | Required by proposed JobHub model; verify customer authority and ATS terms |
| [Lever terms](https://www.lever.co/legal/terms-of-service) | JobHub cache/retention unconfirmed | Public technical access is not a syndication grant | Unconfirmed | Customer access is limited; third-party commercial licence unconfirmed | Obtain employer consent and applicable platform clearance |
| [Workday terms](https://www.workday.com/en-us/legal/site-terms.html) | Separate agreement | Supported export/API agreement required | Contract-specific | API authorization under separate agreement | Employer plus Workday-approved integration |
| [Ashby terms](https://www.ashbyhq.com/resources/terms) / [partner feed](https://developers.ashbyhq.com/docs/dedicated-partner-job-feeds) | Confirm retention in partner/source agreement | C for approved opt-in feed; D for independent public-board aggregation | Not specified in feed guide | Partner commercial terms unconfirmed | Explicit customer opt-in for partner feed |
| [SmartRecruiters partner API](https://developers.smartrecruiters.com/docs/partners-job-board-api) | Confirm partner retention | C for purchased partner publication; public-board mirror D | Contract-specific/unconfirmed | Marketplace posting relationship; not unrestricted reuse | Customer publication purchase or separately authorized board |
| [Workable terms](https://www.workable.com/terms) | Cross-company retention unclear | Own-career-site purpose documented; aggregation D | No public-feed rule established | Customer terms do not establish JobHub commercial rights | Written employer permission plus platform-scope review |
| [Tellent/Recruitee terms](https://recruitee.com/terms) | JobHub retention unclear | Careers integration B; third-party reuse D | Unconfirmed | No blanket commercial feed licence established | Employer-issued Careers token and written display rights |
| [BambooHR developer terms](https://www.bamboohr.com/legal/developer-terms-of-service) | Limited necessary cache; revocation deletion rules | No unrestricted redistribution grant; integration data confidentiality requires explicit scope resolution | Branding/marks need permitted use; no vacancy badge clearance inferred | Data sale and unrelated exploitation restricted | Written authorization + approved third-party auth, not shared customer key |
| [Teamtailor partner docs](https://partner.teamtailor.com/job_boards/) | Retention unconfirmed; remove withdrawn ads | C for activated partner distribution; customer-token aggregation D until agreed | Contract-specific/unconfirmed | Partnership terms required | Customer activation or authorized customer-token integration |
| [Personio terms](https://www.personio.com/terms/) / [XML FAQ](https://support.personio.de/hc/en-us/articles/29375445597725-Frequently-asked-questions-on-XML-job-integration) | No unrestricted cache licence verified | Multiple websites supported, not blanket third-party syndication | Unconfirmed | Commercial marketplace use needs clarification | Employer activation is technical, separate permission still needed |

For every source obtain the rights to store specified fields, display full text
commercially, preserve application/tracking links, use company branding, and
retain/purge content within agreed deadlines. Record the applicable agreement,
version/date, approver, scope and revocation contact. Employer permission does not
override ATS restrictions; ATS partnership does not prove ownership of content.
Default `production_permissions_confirmed=false`. No source is automatically
promoted by a successful dry-run, a public URL or a permission checkbox from an
unverified account.

## 5. Kazakhstan relevance

Targeted searches covered all ten hosted-domain families. Examples below are
employer-board evidence, not statements that these ATS are common in Kazakhstan.
No market-share or obtainable-vacancy count was established. Remote eligibility
requires explicit Kazakhstan inclusion or employer confirmation; “remote,” EMEA,
CIS and a country selector in an application form are insufficient alone.

| ATS | Evidence | Assessment |
| --- | --- | --- |
| Greenhouse | [Wolt board result](https://job-boards.greenhouse.io/wolt/jobs/6967411?gh_src=zw6ay1cr1us) includes Almaty/Astana roles | International employer hiring in KZ; indexed board may replace an old job page |
| Lever | [Aleph board](https://jobs.lever.co/aleph) lists Almaty client-partner roles; [Binance board](https://jobs.lever.co/binance) includes Kazakhstan roles | Direct KZ relevance; candidate eligibility still checked per posting |
| Workday | [Mondelēz accounting role](https://mdlz.wd3.myworkdayjobs.com/en-US/External/job/Analyst--Accounting---External-Reporting--Indirect-Tax_R-166872) names Almaty | Relevant enterprise employers, but supported access is unresolved |
| Ashby | [The Flex engineering posting](https://jobs.ashbyhq.com/the-flex/82eafc9b-c4c0-4310-8b9d-9f5410ff0d53) lists Kazakhstan among remote locations | Possible remote supply; broad location list, not evidence of a local employer base |
| SmartRecruiters | [Nazarbayev University posting](https://jobs.smartrecruiters.com/NazarbayevUniversity1/744000093971049-chair-of-the-department-of-biology-at-the-school-of-sciences-and-humanities); [KIMEP historical posting](https://jobs.smartrecruiters.com/KIMEPUniversity/743999762242834-full-time-faculty-in-management-area-effective-from-january-2022) | Direct Kazakhstan institutions; old postings establish use, not current openings |
| Workable | [Emerging Travel Group role](https://apply.workable.com/emerging-travel-group/j/F4A1262642) lists Kazakhstan and remote work | Relevant remote example |
| Recruitee | [Syrve support role](https://syrve.recruitee.com/o/customer-support-specialist-l1-1) lists Almaty and remote work | Direct geographic relevance |
| BambooHR | No substantiated Kazakhstan vacancy found in scoped public search | **Unknown**, not evidence of absence; no remote eligibility claim |
| Teamtailor | [GUS board](https://gusglobaluniversitysystems.teamtailor.com/jobs?page=3) lists Kazakhstan remote recruitment work | Relevant example, not a volume estimate |
| Personio | [Synology board](https://synology.jobs.personio.com/?language=de) lists a Kazakhstan freelancer role; [Aiphoria board](https://aiphoria.jobs.personio.com/) has historical KZ remote locations | Potential remote supply; reconfirm live eligibility |

## 6. Existing JobHub contract and smallest extension

Verified against these current files, rather than assuming the architecture
proposal is already implemented:

- [Service](../backend/internal/ingestion/service.go): `JoobleSearcher`,
  `RunJooble`, direct `jooble.Normalize`, writes a run before fetching.
- [Imported model](../backend/internal/jobs/model.go): raw fields and updated time;
  no separate imported apply URL, lifecycle, structured salary or company link.
- [Store](../backend/internal/jobs/postgres_store.go): source advisory lock and
  atomic upsert/run completion; current lock starts **after** collection.
- [SQL](../backend/sql/queries/jobs.sql): conflict key `(source, external_id)`,
  observation-time guard; inserts external application and unknown source status,
  sets `apply_url=source_url`, and does not update source status on conflict.
- [Initial supply schema](../backend/migrations/000002_create_vacancy_supply.up.sql):
  existing `job_sources.source` text primary key, permissions/attribution,
  `company_sources`, timestamps and lifecycle columns. No new migration now.

Retain these foundations. Add a neutral collection phase and make `RunJooble` a
compatibility wrapper later; do not duplicate persistence inside ATS adapters.
Conceptual Go contract (design only):

```go
type VacancyProvider interface {
    ValidateSource(SourceConfig) error
    Capabilities() Capabilities
    Collect(context.Context, SourceConfig, CollectOptions) (CollectionResult, error)
}
```

`CollectOptions` contains deadline, request/byte/job budgets and a metered HTTP
transport. `CollectionResult` contains normalized `ImportedJob` values, observed
provider IDs (including identifiable rejected records), validation issues,
request/fetch counts, provider lifecycle signals, scope/config version, collection
start/end, watermark if supplied, and explicit completeness with a reason.
Adapters own pagination and provider-field mapping. The generic service owns
canonical validation, dedupe, run stats, approved freshness and atomic persistence.
A future streaming/page variant can be added when measured boards require it.
Do not introduce it solely for the first small-board POC.

Extend `ImportedJob` only as needed to pass separate source/apply URLs, nullable
company linkage from verified config, lifecycle signals and available provider
timestamps. Structured salary, employment and workplace values map to existing
columns when reliable; raw labels remain available. Existing storage has more
capability than today's importer. Adapt SQL source queries and regenerate sqlc
when implementation is approved. Do not assume an adapter alone enables archive,
reactivation or separate apply links. Do not invent descriptions or times.

Validate safe HTTPS destinations, nonblank identity/title, truthful full/snippet
kind, salary units/currency, and external application method. Sanitize provider
HTML before display. Credentials, applicant data and screening-question payloads
are outside the vacancy contract. Malformed records must not erase an existing
job or allow absence reconciliation to misclassify it as removed.

## 7. Source Registry proposal

Extend the existing registry concept; do not introduce a parallel `sources`
table or change the job uniqueness key. `id` in CLI means existing immutable
`job_sources.source`, for example `greenhouse:source-<opaque-id>`. `provider` is
adapter type, not source identity. One provider can serve many source rows.

| Field/group | Decision and purpose |
| --- | --- |
| `source`, `provider`, `display_name` | Keep existing fields; source ID survives board URL/name changes |
| `market` | Keep; discovery scope such as KZ, not proof of eligibility |
| `company_id` nullable | Future verified single-company link; never inferred by display name |
| `careers_url` | Canonical human URL; preserve observed host/board identity |
| Typed provider config | One `tenant_key`/board token plus only required region/site/language parameters; avoid duplicate synonymous columns |
| `enabled` | Existing kill switch; new ATS entries should start disabled |
| `production_permissions_confirmed` | Existing final visibility gate; only privileged reviewed approval sets it |
| Attribution | Keep text; add approved link/label only when required; no arbitrary HTML |
| `refresh_interval` | Necessary per-source schedule; bounded and jittered |
| `freshness_policy` | Versioned mode, max display age, missing-observation threshold, removal action, purge deadline |
| Permission record reference | Internal approved fields, commercial scope, evidence, approver/time, terms version, revocation/expiry and contact |
| Secret binding | Separate restricted credential store keyed by source; no tokens or secret URLs in registry/public rows |
| Config version | Internal value required to invalidate review/dry-run after board/scope/policy changes |
| Runtime health | Derive from persisted runs first; cache later only if needed; not user-editable configuration |

Do not store both `last_failure` free text and unstructured exception messages.
Use sanitized codes and timestamps. Do not add an independently editable health
state, guessed employer ID, arbitrary endpoint override, or analytics counters
without persisted observations. A minimal first iteration can keep typed config,
permission evidence and policy in restricted server configuration linked to the
existing source key; persistent configuration changes need a separately approved
schema design. No credentials appear in public job/source DTOs.

Use an internal uniqueness check on provider + canonical tenant/region/site and
approved board scope to prevent registering the same board twice. Languages and
KZ display filters should not create independent copies of the same job. A tenant
change to a different company is a new source; a verified alias/rename preserves
identity with a reviewed config update.

### Company relationship and authorization flow

One verified JobHub company may have native jobs and multiple approved sources.
Keep a source unlinked until authority is established. Multi-company/partner
feeds need per-employer identity mappings; do not force the whole feed into one
company. Reuse `company_sources` only with a verified external-company identity;
board slugs and display names alone do not establish legal ownership.

Future flow: Company settings → URL → deterministic detection → verification
and source-use permission → dry-run → review → enable. Existing server-side
company-management checks apply before configuration or secret access. Establish
control through a trusted ATS admin authorization, a company-domain challenge
plus accountable employer review, or an operator-reviewed contractual mandate.
A pasted board URL, matching email suffix or readable token alone is insufficient.
Credential proof and content-display permission are separate checks.

Review the dry-run counts, scope, destination, missing fields, permission terms
and removal policy. A privileged approval records the exact config version;
enabling rechecks that version, source ownership and permission validity.
Revocation disables sync/public display immediately and starts agreed deletion.
No frontend flow, email, ownership claim or company merge is implemented here.

## 8. Deterministic ATS detection

Detection returns `{provider, tenant_config, canonical_careers_url, evidence,
confidence: exact|unknown}`. It does not enable a source or prove ownership.
Parse URLs with a real URL parser; require exact hosts or label-bound suffixes,
not substring matches such as `evilgreenhouse.io` or `lever.co.attacker.example`.
Preserve tenant case and reject credentials, unexpected ports and path traversal.

| Signal | Extract / result |
| --- | --- |
| `boards.greenhouse.io/{board}`, `job-boards.greenhouse.io/{board}` | Greenhouse board; validate regional variants explicitly before adding them |
| `jobs.lever.co/{site}`, `jobs.eu.lever.co/{site}` | Lever site and region |
| `{tenant}.wdN.myworkdayjobs.com/[locale]/{site}` | Workday candidate config with exact host/site; detection does not authorize CXS |
| `jobs.ashbyhq.com/{board}` | Ashby case-preserved board |
| `careers.smartrecruiters.com/{company}`, `jobs.smartrecruiters.com/{company}/…` | SmartRecruiters company identifier |
| `apply.workable.com/{account}` | Workable account candidate; verify account mapping before collection |
| `{tenant}.recruitee.com` | Recruitee tenant |
| `{tenant}.bamboohr.com/careers/…` | BambooHR tenant; supported feed still unresolved |
| `{tenant}.teamtailor.com` | Teamtailor board; API account/stack still needs authorized config |
| `{tenant}.jobs.personio.de` or `.com` | Personio account and domain |

Known-host URL parsing requires no requests. For a custom employer URL, return
unknown unless a future approved bounded inspection finds an explicit supported
ATS link/embed. That optional phase allows one supplied page and at most two
redirect hops, no recursive traversal, no JS/browser execution, no guessed slug
probing. Conflicting ATS evidence returns unknown. Keep discovery and feed health
separate: a valid zero-job board is not failed detection.

Any future outbound request must reject private/loopback/link-local/reserved
addresses, check DNS and every redirect, and restrict provider API hosts. Treat
provider-returned pagination/detail URLs with the same rules. Do not forward
credentials across hosts. Enforce response bytes, timeout and content type; XML
parsing must disable external entities. Unknown hosts/regions require review,
not WAF bypass, stealth or a general crawler.

## 9. Dry-run design

Proposed command, **not available or run in this stage**:

```text
jobhub collect --source <source-id> --dry-run --sample 5
```

Resolve a reviewed source through read-only configuration access. Build the same
adapter, transport, normalizer, canonical validator and deduper used by ingestion.
Do not instantiate a writable store, call `BeginIngestionRun`, update health,
advance cursors, change permissions, populate caches or enqueue notifications.
No database mutation includes audit/run rows. A read-only DB role is preferable
when resolving registered configuration; approved fixture config permits offline
runs with no DB at all. A dry-run still uses network quota unless fixture-only.

Output a bounded, versioned report:

| Output | Meaning |
| --- | --- |
| Provider/source/company | Stable source ID, display name and verified company ID if present |
| Requests made | Every actual attempt, including redirects, retries and detail requests |
| Fetched | Raw entries received, including repeats |
| Normalized | Entries that passed provider mapping before generic validation |
| Malformed | Mapping/parse failures; malformed container fails the run |
| Missing critical fields | Generic rejection counts by field; identify overlap with malformed counts |
| Provider duplicates | Repeated exact `(source, external_id)` keys; unique accepted count separately |
| Sample first N | Stable ID ordering; source/apply URLs, title, raw location, nullable fields, description kind; bounded text |
| Lifecycle | Explicit active/closed/unlisted/expiry counts, unknowns, completeness and scope |
| Limits / outcome | Truncated or exhausted budget, validation issues and `complete`, `partial`, `failed` status |

Never report inserted/updated DB counts as measured in dry-run. Optional read-only
comparison can label them **predicted**; not needed for the first POC. Samples
omit secrets, authorization headers, signed feed URLs, applicant fields and raw
exceptions. Preserve approved public tracking in stored URLs but redact sensitive
query parameters in diagnostics. Never print an API URL containing a credential.
Failure/partial results must be unmistakable and return a non-success CLI status;
a valid empty complete board is successful. Disabled sources may be previewed by
an authorized operator; dry-run is not a permissions bypass.

## 10. Source health and scheduling

Use persisted ingestion runs as the source of truth; add only missing metrics
when implementation is approved. The current schema records basic counters but
not completeness, malformed counts or a source-health roll-up. Proposed additions
are real run observations, not metrics claimed to exist today.

Track `last_attempt_at`, `last_success_at` (successful persistence),
`last_complete_success_at`, last failure time/code/category, consecutive failures,
`jobs_seen_last_run`, request count, normalized/malformed counts, scope/config
version and completeness. Derive counts from finalized runs; a later cache must
be updated transactionally. Never substitute dry-run observations for sync success.

Health is a derived internal view, with precedence:

1. **disabled:** explicit administrative switch is off.
2. **permission_blocked:** source enabled but approval missing/expired/revoked;
   stop scheduled collection/display as applicable. Development preview is a
   separate reviewed operation, not production health.
3. **failing:** repeated execution errors (initial proposal: three consecutive
   failures) or invalid credentials/config requiring operator action.
4. **degraded:** isolated error, partial collection, excessive malformed records,
   missing first sync or overdue expected run. Label reason `never_synced` rather
   than inventing success.
5. **healthy:** latest required collection and persistence succeeded within its
   source-specific schedule; a schema-valid empty response qualifies.

401/403 are access errors, 429 quota/backoff, timeout/5xx transport, invalid JSON/XML
parse/schema, and DB rollback persistence errors. Do not classify every 403 as a
legal revocation or parse HTML login pages as empty boards. A real successful
empty board resets execution failures; an abrupt count collapse can trigger
review without being called fetch failure. Do not update freshness on failed runs.

Schedule enabled, approved sources independently, with jitter and shared provider
rate budgets. One source may have only one active collection generation. Use a
lease/generation check covering fetch through commit; the existing post-fetch
advisory lock alone cannot stop older snapshots reconciling over newer results.
Bound retries for 429/temporary 5xx/network failure, respect `Retry-After` (seconds
or HTTP date), apply jittered backoff, and stop if retry exceeds run budget. Do not
retry auth/schema errors blindly. Attempt metrics must include failed/retried calls.
Operator recovery/outage events should fire on state transitions, not every poll.
No scheduler or notification integration is added now.

## 11. Freshness and deduplication policy

Each source selects a validated provider capability and policy version. Adapter
capability alone does not prove an individual run was complete. A full snapshot
must have exhausted all pages, retained the same source/scope/config, validated
container and identifiers, met total-count checks when available, and avoided
request/job/byte caps, unresolved errors or suspicious redirects. Unidentifiable
malformed entries invalidate reconciliation; identifiable rejected entries count
as seen without overwriting valid stored content. Conflicting duplicate IDs make
completeness uncertain until resolved. Never expire on partially parsed input.

Keep ingestion membership distinct from JobHub display filtering: collect the
whole authorized board and track its IDs, then select KZ-eligible display records.
Changing geography/language filters cannot masquerade as upstream closure. If a
source must use a partial or ranked scope, absence has no lifecycle meaning.
For offset-based feeds, concurrent upstream edits can skip entries even after
pagination; use repeated complete observations rather than claiming atomicity.

Within one transaction, check generation/config version, upsert accepted records,
apply explicit lifecycle signals, reconcile eligible missing IDs, and finalize run
stats/health. Failure rolls back job changes and reconciliation. Store disappearance
as local `removed` evidence; never fabricate `external_archived_at`. Reappearance
of the same provider ID reactivates the same local UUID and retains `first_seen_at`.
The SQL conflict path must explicitly support that reactivation when implemented.

| Policy family | Initial design, subject to source agreement |
| --- | --- |
| Greenhouse | Refresh every 6h; hide after two complete missing observations at least one interval apart; 24h maximum unverified display age |
| Lever | Refresh every 6h; two complete paginated observations; 24h maximum display age; no upstream publication-time invention |
| Ashby public | Listed-only, identity verified first; 6h refresh and 24h age only after fixture validation |
| Ashby partner | Hourly generation; poll no faster than contracted generation; missing/removal semantics agreed before enabling |
| SmartRecruiters | 6h refresh after access clarification; explicit inactive wins; otherwise repeated complete list checks |
| Workable / Recruitee | Start with validated complete published scope; set cadence/age only after public/token contract and fixture review; deltas never reconcile absence |
| Teamtailor | Customer API cadence negotiated within limits; partner XML no faster than its three-daily generation; process explicit removals promptly |
| Personio | Hourly refresh following official guidance; propose 6h max age and two hourly missing observations |
| Workday / BambooHR | No production policy until supported retrieval/publication contract is established |
| Existing Jooble | Preserve ranked/partial behavior; no absence expiry |

These are proposed operational defaults, not vendor SLAs or permission to poll.
Use each agreement's stricter removal/purge obligations. A stale deadline hides
content as unverified, not as provider-confirmed closure. Empty complete snapshots
can contribute to the missing threshold; malformed/failed/limited snapshots cannot.
Do not refresh from old artifacts. A 304 is usable only if a validated retained
snapshot and its permission/scope are still available; otherwise fetch fully.

Retain `UNIQUE(source, external_id)`. Provider posting IDs are opaque lossless
strings; never round numeric JSON through float64, trim meaningful characters,
replace them with requisition IDs, or hash title/company/location. Languages and
locations do not create new identity unless the provider supplies different
posting IDs. Cross-source similarity remains diagnostic; no automatic job or
company merge. Keep reviewed URL aliases separate from the immutable source key.

## 12. Open-source patterns reviewed

Read-only review at the revisions linked below. All three LICENSE files are MIT;
direct reuse would require copyright/licence notices, dependency review and
separate provider-data permission. No code is copied. Their documentation claims
are implementation clues, not authoritative API contracts.

| Project / pinned revision | Useful observed patterns | Adaptation / exclusions for JobHub |
| --- | --- | --- |
| [Esteban-PG/Job-alert-bot](https://github.com/Esteban-PG/Job-alert-bot/tree/6895e99a4853a9e226d0c0af65c494de319c6bd6) ([MIT](https://github.com/Esteban-PG/Job-alert-bot/blob/6895e99a4853a9e226d0c0af65c494de319c6bd6/LICENSE)) | `config.py`/YAML dispatch to fetchers; `storage.py` stores seen IDs and failure/recovery state; CLI dry-run; scheduled workflow serializes runs | Use config-driven adapters and outage transitions. Do not copy SQLite/Actions persistence, country substring matching or notification delivery as ingestion success. Its DB initialization still creates schema, so its dry-run is not JobHub's no-mutation contract |
| [shunsukefuruyama/ats-jobs](https://github.com/shunsukefuruyama/ats-jobs/tree/e0c99d67bced73c295fa33eaef5b566f3ed41ede) ([MIT](https://github.com/shunsukefuruyama/ats-jobs/blob/e0c99d67bced73c295fa33eaef5b566f3ed41ede/LICENSE)) | Provider definitions; hostname detection; pagination; bounded HTTP retries; tests distinguish disappeared board from empty board | Retain transport/adapter split and gone-vs-empty regression fixtures. Reject broad parallel slug probes, unsafe substring host matching and silent job caps as complete snapshots. Do not cap a provider's longer Retry-After into a premature retry |
| [Babak-hasani/company-career-scraper](https://github.com/Babak-hasani/company-career-scraper/tree/af96d4e4673b4075d768f4cf392b86d5a8f1ffc6) ([MIT](https://github.com/Babak-hasani/company-career-scraper/blob/af96d4e4673b4075d768f4cf392b86d5a8f1ffc6/LICENSE)) | ATS function dispatch, board-token configuration, status outcomes, SmartRecruiters pagination | Keep explicit config/status separation. Reject company-name slug guessing, nonempty-result detection as proof, URL-only dedupe, and spreadsheet persistence |

No durable transactional notification outbox was established in the reviewed
paths. For a future need, a Go/PostgreSQL outbox would enqueue after successful
persistence with an idempotent event key and a separate delivery worker; failed
notification delivery must not roll back or duplicate ingestion. This is a JobHub
proposal, not an attributed feature of those projects. Telegram work remains out
of scope. The bounded review does not certify these projects as production-ready
or justify adding another collector dependency.

## 13. First and second adapter decision

**First: Greenhouse.** Public documented GETs, explicit posting identity,
whole-board descriptions, useful update/deadline fields and KZ employer evidence
make it a small, inspectable first extension. Legal status is conditional, not
better merely because authentication is absent. Employer permission and applicable
ATS terms must be resolved before live POC/production use. Missing employment or
remote metadata stays unknown; quality should not be manufactured for a richer UI.

**Second: Lever.** It adds relevant KZ hiring, clear posting IDs, explicit apply
URLs and structured workplace/salary where supplied. Pagination and missing
public update/closure events add modest work. It is next after Greenhouse proves
the shared source, dry-run and reconciliation boundaries.

SmartRecruiters has particularly useful local-university evidence but current
auth ambiguity and separate distribution programme add onboarding questions.
Ashby partner feeds offer clearer opt-in distribution but require provisioning;
public identity documentation needs clarification. Workday's KZ reach does not
outweigh access/contract uncertainty for the first adapter. Do not start ten
adapters concurrently. Revisit priority if an authorized KZ employer supplies a
different supported feed with clear rights; do not implement speculatively.

## 14. Bounded Greenhouse POC proposal — not executed

After explicit implementation approval and source permissions:

- Select **two employer-authorized small boards**, optionally a third controlled
  test board, using documented public career URLs. The documented Greenhouse
  example token `example` is a schema example, not assumed live. Wolt is a KZ
  relevance lead, not an authorized test fixture or a small-board guarantee.
  Final board names and permission records must be reviewed before collection.
- Limit accepted/processed inventory to **200 jobs total** (target 100–200),
  **12 HTTP attempts total** across initial dry-run and two syncs, including retry,
  redirects and at most two optional detail probes. At most one request/second,
  one in flight, 20-second request timeout and 5 MiB response limit are proposed
  local limits, not provider quotas. Choose boards small enough for full snapshots.
- Greenhouse does not document list pagination: a 200-job processing limit cannot
  guarantee the server transmits at most 200 jobs. Abort an oversized board and
  classify it partial; do not silently take 200 and reconcile. Validate board size
  with the employer first; keep the byte cap as the network bound.
- Run dry-run first with samples and explicit completeness. Then use a deliberately
  isolated development PostgreSQL database and development-only source keys;
  keep production permission false. Never reuse remote/staging credentials.

| Proof | Acceptance criterion |
| --- | --- |
| Fetch → normalize | Bounded complete list, truthful fields, errors distinguished from empty |
| Stable identity | Exact posting ID retained as text; source key scoped to board |
| Repeat sync | Same IDs keep local UUID and first-seen time; no extra rows; a controlled fixture edit updates the existing row |
| Closure / disappearance | Controlled offline response removes an ID in two qualifying snapshots; then restores it. Missing job hides/reactivates correctly; no claim a live job closed unless observed |
| Failure safety | Timeout, invalid JSON, truncated body, partial scope, malformed ID and stale generation cause no false freshness or removal |
| Application | Original `absolute_url` preserved; `application_method=external`; imported job never offers native Apply |
| Permission | Unconfirmed/disabled source absent from production-mode list/count/detail; dry-run cannot change approval |
| No writes in dry-run | Read-only role succeeds; jobs, runs, health and cursor state unchanged |

Use contract fixtures for deterministic closure, deadline and reappearance rather
than changing a real employer's vacancies. Use existing integration boundaries
for transactional and external-CTA checks; any later browser QA is only kk/ru.
Reconciliation tests must include a successful empty board and overlapping runs.
Report request totals and results before considering production. Stop if board
permissions, complete scope or stable identity cannot be proven; don't expand the
POC request allowance or switch to scraping.

## 15. Questions requiring confirmation

1. For each employer: who can authorize JobHub, which legal entities/boards are
   included, and can JobHub store, commercially display and index full descriptions?
   Which fields, logos, attribution, tracking, retention and removal SLA apply?
2. For Greenhouse/Lever: do applicable customer/partner terms cover this employer-
   authorized marketplace display, and what GET limits apply? Is separate platform
   approval required? Which two small boards can be approved for the POC?
3. For Workday: which supported export/API channel and agreement cover job-board
   distribution, and what are posting identity, completeness and removal semantics?
4. For Ashby: is a stable standalone ID contract available for the public feed,
   or should JobHub use the opt-in partner feed? Confirm partner retention,
   commercial terms, identity continuity and withdrawn-employer removal.
5. For SmartRecruiters: confirm anonymous external-only reads versus the newer
   auth language, anonymous rate scope, and whether JobHub should onboard as a
   marketplace job-board partner.
6. For Workable/Recruitee: obtain current response schemas, public-field guarantees,
   token rollout requirements, ID stability, paging/completeness and rate limits.
7. For BambooHR: establish an approved third-party auth flow and public-job-only
   data contract, including full descriptions, URLs and disclosure rights. Do not
   accept a customer API key as a workaround.
8. For Teamtailor/Personio: confirm distribution/retention rights, account or
   subsidiary mapping, feed removal obligations, regional hosts and cadence.
9. For every remote role: does the employer actually accept Kazakhstan-based
   candidates, and do location tags describe work location, relocation sourcing,
   or merely market coverage?

Implementation remains pending approval. This document grants no new API access,
production permission or authorization to run the proposed POC.
