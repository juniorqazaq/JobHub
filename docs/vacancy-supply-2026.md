# Vacancy supply strategy, 2026

Research date: 24 September 2026. This document is a current-source review,
not permission to enable a provider in production. Previous MVP 0 research is
historical context only.

## Executive summary

JobHub already supports native vacancies and a bounded Jooble Kazakhstan import.
The database and public API preserve source identity, use an external application
CTA for imports, deduplicate within a provider, and can hide sources whose
production permissions are not confirmed. The remaining constraint is legitimate,
fresh supply rather than basic ingestion mechanics.

The recommended next source to investigate is **Enbek**, through a written
integration/redistribution agreement and a provider-supplied working feed. Enbek
is the strongest Kazakhstan-specific source in this review: its official portal
says data is updated daily from employers, the state vacancy database, private
employment agencies, media and online platforms. Official statistics report
1.328 million vacancy publications in 2025, and 108,700 publications in June
2026. Those are flow figures, not simultaneously active or legally obtainable
records. They do not support a guaranteed import volume.

Enbek's official web-services page still advertises Jooble- and Yandex-format XML
vacancy feeds, but both linked endpoints returned HTTP 404 on the research date.
No official public licence covering storage, public redistribution, commercial
display, attribution, retention and removals was found. Enbek is therefore class
**C: written agreement/partnership required**, not an implementation-ready API.

Jooble remains class **B for the bounded development POC** and class C for
production until its API-specific persistence, redistribution, attribution,
retention and removal rules are confirmed in writing. No Jooble API requests
were used for this research.

No defensible exact count of obtainable jobs exists yet. A 500–1,000 active-job
target is technically modest relative to Enbek's published market flow and
Jooble's public Kazakhstan inventory claim, but permission, feed operation,
coverage and freshness must be proven before treating any portion as obtainable.

## Decision classes

- **A — Suitable for production investigation:** official access and a use case
  compatible with JobHub, subject to normal due diligence.
- **B — Suitable only for a bounded POC:** enough technical and display clarity
  to test, but production rights or sustainable access are incomplete.
- **C — Requires written agreement/partnership:** technically plausible, but the
  proposed public aggregation, storage or scale is not covered by public terms.
- **D — Unsuitable for JobHub vacancy supply:** no applicable retrieval channel,
  prohibited use, or no verified Kazakhstan coverage.

## Provider decision matrix

### Access, coverage and data

| Provider | KZ coverage | Official access and auth | Free quota / limits | Pagination and incremental sync | Fields, description, employer, salary, location | Timestamps and lifecycle | Class |
| --- | --- | --- | --- | --- | --- | --- | --- |
| **Enbek** | Strong, nationwide | Official page advertises Jooble/Yandex XML feeds; both linked endpoints returned 404. No current auth documentation found. | No feed quota published. Portal use is free, but that is not an export licence. | XML feeds imply snapshots; no documented cursor, delta, page or deletion contract found. | Feed schemas are named, but a current response could not be inspected. Do not assume fields from a dead endpoint. | Portal vacancies publish for up to one month and then leave public view for an employer archive; feed removal semantics are undocumented. | **C** |
| **HeadHunter / hh.kz** | Strong | Official `api.hh.ru`; register an application/API key, OAuth for protected methods. | No current guaranteed free extraction allowance verified. Search depth is capped at 2,000 results. | `page`, `per_page` (up to 100), maximum result depth 2,000; no global archive/delete stream for a third-party mirror. | Rich vacancy detail, description, employer ID/name, structured areas and salary, employment/work formats, URLs. | `published_at`, `created_at`, `archived` and detail state exist; public search absence is not a deletion event. | **C** |
| **Jooble KZ** | Explicit regional KZ API key and dataset | REST POST; country-specific key is embedded in the URL path. | 500 requests total per key, lifetime rather than monthly. Current JobHub POC additionally caps runs at 50 and defaults lower. | `page`, `ResultOnPage`, `totalCount`; no documented delta cursor or removal feed. | ID, title, location text, snippet, salary text, upstream source, type, link, company name, updated time. No full detail or stable employer ID documented. | `updated` only; no published, expiry, archived or deleted signal in the documented result. | **B POC / C production** |
| **Adzuna** | No Kazakhstan market was verified in official country/API material | REST search with `app_id` and `app_key`. | 25/min, 250/day, 1,000/week, 2,500/month by default. | Page in URL plus `results_per_page`; no deletion stream documented. | ID/title, snippet only, company display name, hierarchical location/coordinates, salary bounds and predicted flag, category, contract fields, redirect URL. | `created`; no public archive/expiry/delete signal established. | **D for KZ now** |
| **LinkedIn** | Jobs exist in KZ, but there is no suitable public retrieval API | Approved Talent Solutions partner APIs and XML flows send employer/ATS jobs **into** LinkedIn. OAuth/client credentials and signed agreement required. | Partner contract limits; not a public free search API. | Posting APIs support create/update/renew/close, not public marketplace retrieval. | Rich outbound posting schema, but irrelevant as an inbound supply API. | Lifecycle exists for jobs a partner submits to LinkedIn. | **D for retrieval** |
| **QSamruk** | Strong but limited to Samruk-Kazyna group; public site displayed 1,012 vacancies on the research date | No official public API/feed documentation found. | Not documented. | Not documented. | Public website has job/company data; HTML is not an approved feed. | Public lifecycle exists on the site, but no machine-readable contract was found. | **C** |
| **Rabota.kz** | Kazakhstan-focused | No official public vacancy API/feed documentation was found; the site was not used as a data source. | Not documented. | Not documented. | Unknown for an official integration. | Unknown. | **C** |
| **Greenhouse employer feed** | Only employers using Greenhouse; KZ volume unknown | Public Job Board GET API, no auth, keyed by an employer board token. | No public quota documented in the reviewed page. | Returns an employer board snapshot; no cursor or tombstone feed documented. | Published job ID, title, full content on request, office/department, URL, language, updated time; detail includes first-published/deadline and optional pay ranges. | Removal can be inferred only from a complete authorized board snapshot; explicit tombstones are absent. | **A with employer authorization / C for broad aggregation** |
| **Lever employer feed** | Only employers using Lever; KZ volume unknown | Public Postings API GET per employer site; application POST requires a key. | Retrieval quota not stated in the official repository documentation reviewed. | `skip`/`limit`; published board snapshot, no tombstone stream. | ID, title, country, locations/team/department, full HTML/plain descriptions, requirements/benefits lists, hosted/apply URLs, workplace type, optional structured salary. | No publication/update timestamp documented in the reviewed posting object; disappearance is meaningful only for a complete authorized board. | **A with employer authorization / C for broad aggregation** |
| **Workable employer feed** | Only employers using Workable; KZ volume unknown | Public published-job endpoints; richer SPI uses employer token with `r_jobs`. | No public quota documented in the reviewed help page. | Account-scoped snapshot; no public incremental cursor documented. | ID, title, department, URLs, structured locations/workplace type, salary, created time; detail can include full description. | SPI states include draft, published, closed and archived; display must filter to published. | **A with employer authorization / C for broad aggregation** |
| **SmartRecruiters employer/partner feed** | Only participating employers; KZ volume unknown | Public per-company Posting API; multi-employer Job Board API requires a Partner API Key and purchased publication relationship. | Customer API: 10 requests/s and 8 concurrent requests. | Public API uses offset/limit; partner publications API supports delivery/status workflow. | Company/posting IDs, title, structured location, employment/experience, released date, full detail sections and apply URL. | Public list returns active postings; partner publication workflow can update delivery status. | **A for authorized employer feed / C for marketplace partnership** |

### Rights, display and production conditions

| Provider | Storage/cache | Public redistribution/display | Commercial restrictions | Attribution | Retention/removal | Production requirement |
| --- | --- | --- | --- | --- | --- | --- |
| **Enbek** | Unclear | Unclear; public access and a listed feed do not establish republication rights. | Unclear | Unclear | Portal says listings leave public view after up to one month; feed duties are unclear. | Written scope plus working endpoint/schema, quota/SLA and removal procedure. |
| **HeadHunter** | Employment-related storage is constrained by API terms. | Standard terms prohibit transferring API data to third-party services for use; a public JobHub database is not covered. | Customer terms also restrict commercial use and third-party transfer. | No badge can substitute for permission. | Terms require material to remain current and archived vacancies to be removed promptly. | Bespoke written agreement for public aggregation and persistence. |
| **Jooble KZ** | API-specific persistence/retention rules were not located. | KZ API page explicitly says portals/search engines may publish responses in their own design. | Exact production/commercial boundary remains unclear. | No definitive API-specific badge/text requirement found. | No documented expiry/removal feed or retention period found. | Written answers on storage, redistribution, attribution, upstream rights, retention and removal; sustainable quota. |
| **Adzuna** | Publishing listings is permitted; other ongoing aggregation/research may require a licence after a 14-day validation period. | Explicitly permitted for listings when obligations are met. | Licence/written consent may be required outside permitted listing publication and for ongoing organizational use. | “Jobs by Adzuna” linked treatment, at least 116×23 px; separate Jobsworth attribution. | On termination, remove acquired data and insertion code from site pages immediately. | Verified KZ coverage and any required commercial licence/increased quota. |
| **LinkedIn** | Only under partner agreement for approved flows. | No public retrieval/redistribution permission. User Agreement prohibits scraping/copying and display/distribution without consent. | Partner-only Talent products. | Contractual. | Contractual. | Not a JobHub inbound source; no POC. |
| **QSamruk / Rabota.kz** | Unclear | Unclear | Unclear | Unclear | Unclear | Direct partnership and documented feed terms. |
| **ATS employer feeds** | Public technical access is designed mainly for the employer's own career site. Cross-company retention is not granted merely by public endpoints. | Use only for an employer that authorizes JobHub, or under the ATS job-board partner programme. | Productized/multi-customer access often requires OAuth/partner status or contract. | Follow the employer/ATS agreement; no universal badge rule was found. | Reconcile complete employer boards; hide missing/closed jobs conservatively and purge as contracted. | Signed employer authorization or platform partner agreement, stable employer identity and removal contact. |

## Provider-specific findings and official evidence

### Enbek

The [official portal description](https://www.enbek.kz/ru/node/3205) says vacancy
data across Kazakhstan is updated daily from employers, the state database,
private agencies, media and online employment platforms. The
[web-services page](https://www.enbek.kz/ru/web-services) lists
[Jooble-format XML](https://www.enbek.kz/ru/xml/jooble) and
[Yandex-format XML](https://www.enbek.kz/ru/xml/yandex). Both feed URLs returned
HTTP 404 on 24 September 2026. A web page naming a feed is evidence of intended
technical distribution, but it is not evidence that the endpoint works or that
JobHub may store and republish it commercially.

The [official FAQ](https://www.enbek.kz/ru/faq) states that a vacancy is public
for up to one month and then becomes unavailable to candidates while remaining
in the employer's archive. This is a useful lifecycle signal only if the future
feed reliably reflects it. No official API replacement, authentication guide,
quota, open-data dataset, reuse licence, attribution rule or removal SLA was
found. The Kazakhstan open-data search did not establish a current vacancy
dataset suitable for this integration.

Official statistics show scale, not obtainable inventory. The Ministry reported
[1.328 million vacancies published in 2025](https://www.gov.kz/memleket/entities/enbek/press/news/details/1150201?lang=ru)
and [108,700 in June 2026](https://www.gov.kz/memleket/entities/enbek/press/news/details/1258464?lang=ru).
These counts can include turnover and republication; they are not active snapshot
counts and cannot be used as an import forecast.

### HeadHunter / hh.kz

The [official OpenAPI](https://api.hh.ru/openapi/en/redoc) provides rich vacancy
search/detail data and identifies the shared API used by hh.kz. Search pagination
is capped at a depth of 2,000 results. Technically, HH would be high quality.

The blocker is use rights. The [API terms](https://hh.ru/article/15116) require
application registration, restrict use to employment purposes, require current
materials and prompt deletion of archived vacancies, and prohibit transferring
API database data to third-party services for use. This makes a persistent public
JobHub vacancy database unsafe under standard terms. Do not build an HH importer
without a written agreement that expressly covers JobHub's storage, public
display, employer data, cache duration and archive/removal process.

### Jooble Kazakhstan

The [Kazakhstan API page](https://kz.jooble.org/api/about) expressly describes
portals/search engines querying Jooble and publishing responses in their own
design. The current [REST documentation](https://help.jooble.org/en/support/solutions/articles/60001448238)
says each country requires its own key and the free plan has a lifetime limit of
500 requests per key. It documents keyword/location search, pagination and the
fields shown in the matrix. The [connection guide](https://help.jooble.org/en/support/solutions/articles/60000922689-how-to-connect-to-the-jooble-rest-api)
puts the key in the URL path, which is why request URLs must never be logged.

The public material reviewed does not settle database persistence, cache duration,
redistribution beyond rendering a response, attribution, upstream publisher
rights, retention after termination, or the required reaction to removed jobs.
There is no documented detail/delete endpoint. JobHub's existing 72-hour default
freshness is a conservative local POC policy, not a Jooble lifecycle guarantee.
Production permissions must remain false.

### Adzuna

The [search documentation](https://developer.adzuna.com/docs/search) exposes a
paged job-ad search and only a description snippet. The
[API terms](https://developer.adzuna.com/docs/terms_of_service) explicitly permit
publishing listings, specify default quotas and attribution, and require removal
of acquired data from pages on termination. This is clearer than most providers,
but no official Kazakhstan API market/local domain was verified. Adzuna is not a
current KZ supply candidate unless the provider confirms coverage and licensing.

### LinkedIn

LinkedIn's [Job Posting overview](https://learn.microsoft.com/en-us/linkedin/talent/job-postings/job-posting-overview?view=li-lts-2026-03)
describes authorized clients, ATS vendors and distributors sending jobs to
LinkedIn. It does not provide a public job-search/retrieval feed for JobHub.
Access requires partner approval and an API agreement. The
[LinkedIn User Agreement](https://www.linkedin.com/legal/user-agreement) prohibits
scraping/copying the service and displaying or distributing obtained information
without the content owner's consent. LinkedIn is therefore unsuitable as inbound
vacancy supply.

### Kazakhstan-focused boards

[QSamruk](https://qsamruk.kz/) displayed 1,012 vacancies on the research date,
but no official public API/feed and reuse terms were found. Treat this as a
potential partnership lead, not extractable inventory. The same conclusion
applies to Rabota.kz: Kazakhstan relevance alone does not authorize an HTML
scraper or public mirror.

### Employer and ATS feeds

Employer-authorized feeds are the clearest production path after Enbek outreach.
They trade aggregate volume for reliable provenance and employer consent:

- [Greenhouse Job Board API](https://docs.greenhouse.io/job-board.html) exposes a
  company's published jobs publicly without GET authentication and can return
  full content, timestamps and optional pay data.
- [Lever Postings API](https://github.com/lever/postings-api) exposes complete
  public postings with hosted/apply URLs, structured categories and optional
  salary data.
- [Workable careers API guidance](https://help.workable.com/hc/en-us/articles/115012771647-Using-the-Workable-API-to-create-a-careers-page)
  provides public published-job endpoints and richer employer-authorized API
  access with explicit draft/published/closed/archived states.
- [SmartRecruiters Posting API](https://developers.smartrecruiters.com/docs/endpoints)
  exposes active postings by company. Its
  [Job Board API](https://developers.smartrecruiters.com/docs/partners-job-board-api)
  is the correct multi-employer route and requires a partner API key and posting
  relationship.

Public endpoint availability is not blanket aggregation permission. JobHub should
import an ATS board only after the employer authorizes display and storage, or
after the ATS approves JobHub as a job-board partner. The default CTA should stay
external unless an application integration is separately approved.

## Recommended next source: Enbek partnership investigation

Enbek is the best next source to investigate technically because it has the
strongest Kazakhstan-specific coverage, authoritative provenance, an explicit
public lifecycle of up to one month and an official page that already describes
machine-readable feeds. It could materially move JobHub toward 500–1,000 fresh
vacancies without combining many small sources.

The limitations are decisive:

1. Both advertised vacancy feed endpoints currently return 404.
2. No public feed authentication, quota, pagination/snapshot or deletion contract
   was found.
3. No public licence expressly permits JobHub to persist, normalize, commercially
   display and redistribute vacancy/employer content.
4. Annual/monthly publication counts do not equal active feed size.
5. A full snapshot may contain vacancies aggregated from third parties whose
   downstream rights differ from employer-originated Enbek vacancies.

### Questions for Enbek / АО «Центр развития трудовых ресурсов»

Ask for written answers before coding:

1. What replaces `/ru/xml/jooble` and `/ru/xml/yandex`, and is there a test feed?
2. Is the feed a full active snapshot or a delta? How are update, archive, expiry,
   deletion and republication represented?
3. May JobHub store and publicly display titles, descriptions, employer names,
   salary and location? May it normalize/search/filter those fields?
4. Are third-party-origin vacancies included, and does Enbek grant downstream
   display rights for them?
5. What attribution, backlink, logo and source naming are mandatory?
6. What cache/retention period and deletion SLA apply, including termination?
7. Is commercial display permitted? Is a contract, partner registration or fee
   required?
8. What authentication, request quota, refresh interval, IP rules and SLA apply?
9. Are stable vacancy and employer identifiers guaranteed across updates?
10. Is external linking required, and may JobHub ever accept applications?

## Proposed bounded POC after written approval

Do not start the POC until Enbek supplies a working endpoint/sample and written
permission for the test. Then use a **maximum of 10 feed requests**:

1. One schema/sample validation request.
2. Up to three snapshot or page requests for a narrowly agreed Kazakhstan scope.
3. One repeat request to prove stable IDs and idempotent upsert.
4. One controlled provider update/removal observation if Enbek supports it.
5. Four requests reserved for documented retries or pagination validation.

Use no HTML scraping. Store at most 100 records in an isolated development
database, keep external CTAs, and log counts without logging credentials or
credential-bearing URLs.

### Stop/go criteria

Proceed to implementation only when all are true:

- written display, storage, commercial-use, attribution, retention and removal
  rights match JobHub's intended behavior;
- a stable official endpoint and schema are available;
- stable vacancy identity and a reliable archive/removal or complete-snapshot
  rule are documented;
- quota and refresh cadence can keep jobs fresh at the intended volume;
- sample data has usable title, employer/source, location, content and CTA fields;
- the provider confirms which upstream records JobHub may redistribute.

Stop if permission excludes public display/persistence, the feed cannot represent
active lifecycle safely, credentials must reach the browser, or the achievable
refresh rate would leave stale vacancies public.
