# MVP 0: official vacancy supply research

Status: proposal only; checked 2026-09-23. No integration, migrations, provider
accounts, credentials, scraping, or product features were implemented.
[MVP 1](mvp1-api-contract.md) remains unchanged and deferred.

The repository currently has only the infrastructure schema migration, not
implemented jobs/company tables. The companion [schema proposal](mvp0-source-schema.md)
extends the planned domain model, not a deployed database.

## Recommendation

Choose **Jooble Kazakhstan for one bounded proof of concept**, conditional on
obtaining its regional key and confirming API-specific storage, commercial
display, attribution, retention and removal permissions. Its official API is
intended for integrating vacancies into another platform, and it has a Kazakhstan
registration portal. This is a candidate selection, not a statement that public
republication rights or sustainable production access have been secured.

Start with an explicitly limited request budget, for example 50 requests, and a
few agreed searches. Measure relevance, duplicates, missing fields and freshness.
Do not describe this sample as the complete Kazakhstan market. No key registration
or provider contact has been performed in this research.
# MVP 0: official vacancy supply research

Status: proposal only; checked 2026-09-23. No integration, migrations, provider
accounts, credentials, scraping, or product features were implemented.
[MVP 1](mvp1-api-contract.md) remains unchanged and deferred.

The repository currently has only the infrastructure schema migration, not
implemented jobs/company tables. The companion [schema proposal](mvp0-source-schema.md)
extends the planned domain model, not a deployed database.

## Recommendation

Choose **Jooble Kazakhstan for one bounded proof of concept**, conditional on
obtaining its regional key and confirming API-specific storage, commercial
display, attribution, retention and removal permissions. Its official API is
intended for integrating vacancies into another platform, and it has a Kazakhstan
registration portal. This is a candidate selection, not a statement that public
republication rights or sustainable production access have been secured.

Start with an explicitly limited request budget, for example 50 requests, and a
few agreed searches. Measure relevance, duplicates, missing fields and freshness.
Do not describe this sample as the complete Kazakhstan market. No key registration
or provider contact has been performed in this research.

Why not the alternatives: HeadHunter has richer structured data but its baseline
terms create a substantial obstacle to the proposed persistent public database;
Adzuna Kazakhstan support could not be verified; Enbek's published feed links did
not return usable feeds during this check.

## Provider verification

### HeadHunter / hh.kz

**Availability and Kazakhstan:** official REST API, shared base `api.hh.ru`.
Use Kazakhstan `area=40` (recheck the region directory) and `host=hh.kz`;
host selection alone is not a geographic filter. Search and vacancy/employer
detail endpoints exist. Current documentation describes OAuth and application
identification headers. Register the application; do not depend on anonymous
access for production.

**Pagination/data:** zero-based `page`, `per_page <= 100`, maximum search depth
2,000. Fields include vacancy ID, title, snippets/detail description, region,
employment/work format, employer ID/name/logos, landing URL, published/created
timestamps, structured salary and archive state. Use current salary/work-format
fields and handle deprecated legacy fields explicitly. Public expiry/deletion
coverage is not guaranteed; private employer archive endpoints are not a global
removal feed. [Official OpenAPI](https://api.hh.ru/openapi/en/redoc).

**Quota:** no current guaranteed free daily request allowance was verified.
A [provider announcement](https://hh.ru/article/22812) describes vacancy viewing
as free, but does not establish a present unlimited extraction entitlement.

**Commercial/display/attribution:** standard terms require application registration
and allow charges for some services. Clauses 4.3 and 4.6 restrict building another
database for third-party access and transferring received data. Clause 3.11
requires prompt removal of archived jobs and restricts altering materials.
Trademark use is restricted; a generic attribution badge is not a substitute
for permission. No blanket approved logo/backlink formula was verified.

**Assessment (inference):** JobHub's public imported database appears to conflict
with these baseline restrictions. A written agreement covering this use would
be needed before selecting it for public supply.
[Official API terms](https://hh.ru/article/15116).

### Jooble Kazakhstan

**Availability/auth:** official REST search; obtain a Kazakhstan-specific key
through the [Kazakhstan API portal](https://kz.jooble.org/api/about). Keys are
regional; keep the key-bearing request URL server-side and redact it in logs.
The [connection guide](https://help.jooble.org/en/support/solutions/articles/60000922689-how-to-connect-to-the-jooble-rest-api)
explicitly describes integration into another platform.

**Free quota/pagination:** 500 requests per key **over its lifetime**, not monthly.
JSON POST search; required keywords/location; page defaults to 1; `ResultOnPage`
controls size (documented maximum not found). Response provides `totalCount`.
No documented job-detail or deletion feed was found. Published/expiry/archive
metadata and structured employer identifiers are absent from the reviewed
response specification. See the companion mapping for available fields.
[REST documentation, updated August 16, 2026](https://help.jooble.org/en/support/solutions/articles/60001448238).

**Commercial/display/attribution:** the connection guide references API terms but
the reviewed public material did not establish exact API-specific caching,
commercial redistribution, retention or branding rules. The general Kazakhstan
terms restrict republication without express written permission (§4(i)).
Do not infer an unrestricted licence from receiving a key. Confirm API terms
override or expressly permit this use, including upstream content rights.
[Kazakhstan terms](https://kz.jooble.org/info/terms).

**Limitations:** search coverage is a sample; continuous refresh needs a separate
quota arrangement. Missing expiry signals require a conservative local freshness
policy approved against provider terms. No authenticated Kazakhstan response was
tested because no key was requested or supplied.

### Adzuna

**Availability/auth:** official REST API; `app_id` and `app_key` query parameters;
Why not the alternatives: HeadHunter has richer structured data but its baseline
terms create a substantial obstacle to the proposed persistent public database;
Adzuna Kazakhstan support could not be verified; Enbek's published feed links did
not return usable feeds during this check.

## Provider verification

### HeadHunter / hh.kz

**Availability and Kazakhstan:** official REST API, shared base `api.hh.ru`.
Use Kazakhstan `area=40` (recheck the region directory) and `host=hh.kz`;
host selection alone is not a geographic filter. Search and vacancy/employer
detail endpoints exist. Current documentation describes OAuth and application
identification headers. Register the application; do not depend on anonymous
access for production.

**Pagination/data:** zero-based `page`, `per_page <= 100`, maximum search depth
2,000. Fields include vacancy ID, title, snippets/detail description, region,
employment/work format, employer ID/name/logos, landing URL, published/created
timestamps, structured salary and archive state. Use current salary/work-format
fields and handle deprecated legacy fields explicitly. Public expiry/deletion
coverage is not guaranteed; private employer archive endpoints are not a global
removal feed. [Official OpenAPI](https://api.hh.ru/openapi/en/redoc).

**Quota:** no current guaranteed free daily request allowance was verified.
A [provider announcement](https://hh.ru/article/22812) describes vacancy viewing
as free, but does not establish a present unlimited extraction entitlement.

**Commercial/display/attribution:** standard terms require application registration
and allow charges for some services. Clauses 4.3 and 4.6 restrict building another
database for third-party access and transferring received data. Clause 3.11
requires prompt removal of archived jobs and restricts altering materials.
Trademark use is restricted; a generic attribution badge is not a substitute
for permission. No blanket approved logo/backlink formula was verified.

**Assessment (inference):** JobHub's public imported database appears to conflict
with these baseline restrictions. A written agreement covering this use would
be needed before selecting it for public supply.
[Official API terms](https://hh.ru/article/15116).

### Jooble Kazakhstan

**Availability/auth:** official REST search; obtain a Kazakhstan-specific key
through the [Kazakhstan API portal](https://kz.jooble.org/api/about). Keys are
regional; keep the key-bearing request URL server-side and redact it in logs.
The [connection guide](https://help.jooble.org/en/support/solutions/articles/60000922689-how-to-connect-to-the-jooble-rest-api)
explicitly describes integration into another platform.

**Free quota/pagination:** 500 requests per key **over its lifetime**, not monthly.
JSON POST search; required keywords/location; page defaults to 1; `ResultOnPage`
controls size (documented maximum not found). Response provides `totalCount`.
No documented job-detail or deletion feed was found. Published/expiry/archive
metadata and structured employer identifiers are absent from the reviewed
response specification. See the companion mapping for available fields.
[REST documentation, updated August 16, 2026](https://help.jooble.org/en/support/solutions/articles/60001448238).

**Commercial/display/attribution:** the connection guide references API terms but
the reviewed public material did not establish exact API-specific caching,
commercial redistribution, retention or branding rules. The general Kazakhstan
terms restrict republication without express written permission (§4(i)).
Do not infer an unrestricted licence from receiving a key. Confirm API terms
override or expressly permit this use, including upstream content rights.
[Kazakhstan terms](https://kz.jooble.org/info/terms).

**Limitations:** search coverage is a sample; continuous refresh needs a separate
quota arrangement. Missing expiry signals require a conservative local freshness
policy approved against provider terms. No authenticated Kazakhstan response was
tested because no key was requested or supplied.

### Adzuna

**Availability/auth:** official REST API; `app_id` and `app_key` query parameters;
country-specific `/v1/api/jobs/{country}/search/{page}`.
[Official overview](https://developer.adzuna.com/overview).

**Kazakhstan:** not verified. The public reference did not establish a supported
`kz` dataset; the interactive documentation exposed no usable country list in
this check. A [2022 market expansion announcement](https://www.adzuna.com/blog/adzuna-is-live-across-20-countries-globally/)
is not evidence of present Kazakhstan API coverage. Do not invent `/jobs/kz` or
equate jobs mentioning Kazakhstan in another market with a Kazakhstan feed.

**Pagination/data:** page in URL, starting at 1; `results_per_page`; maximum not
verified. Search exposes ID/title, description snippet, created timestamp,
redirect URL, company display name, location hierarchy/coordinates, category,
contract type/time and salary bounds with a predicted-salary flag. Stable company
ID, full description, public expiry/archive/deletion feed are not established by
the reviewed search response. Currency/period/gross must not be guessed.
[Search reference](https://developer.adzuna.com/docs/search).

**Quota/terms/attribution:** default limits are 25/minute, 250/day, 1,000/week and
2,500/month; all apply. Publishing listings is an expressly permitted use.
Other commercial, government or academic uses get a 14-day validation period;
ongoing research/aggregation may require written consent/licensing. This is not
a blanket 14-day restriction on permitted advert publishing. Each displayed ad
requires the linked “Jobs by Adzuna” treatment, at least 116 × 23 px, using the
Adzuna logo and relevant local domain. Jobsworth estimates have separate
attribution. Termination requires removing acquired data from website pages.
[API terms](https://developer.adzuna.com/docs/terms_of_service).

### Additional Kazakhstan lead: Enbek

The official [frames and feeds page](https://www.enbek.kz/ru/web-services) links
vacancy XML in Jooble and Yandex formats. Both listed endpoints
([Jooble format](https://www.enbek.kz/ru/xml/jooble),
[Yandex format](https://www.enbek.kz/ru/xml/yandex)) returned 404 during this check.
This does not establish that feeds are permanently discontinued.

Kazakhstan relevance is clear; operational availability, authentication, free
quota, pagination, exported fields/salary/company data, redistribution and
attribution permissions remain unverified. Official
[employer guidance](https://www.enbek.kz/kk/advice/4169) describes a 30-day vacancy
listing period and archival, but this does not establish XML removal semantics.
Do not infer an open-data licence from free public portal access. Ask Enbek for
the current official export/partnership route before considering it as supply.

## Approval boundary and next-phase questions

Approve provider choice and schema direction first. A later POC should establish:

1. Written permission for the exact display/storage model, attribution, permitted
   transformations, retention and removal schedule; any upstream restrictions.
2. A Kazakhstan key and verified regional endpoint, kept only on the server.
3. Available quota, paid expansion, maximum page size and rate-limit behavior.
4. Actual Kazakhstan quality, stable IDs and URLs, timestamp timezone, update
   semantics, archive handling and acceptable freshness window.
5. Idempotent repeat import and honest outbound-only display, without suggesting
   an application was submitted through JobHub.

This research used official documentation and officially linked feed endpoints.
It did not collect website listings or exercise authenticated provider APIs.
