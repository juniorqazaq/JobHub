# Kazakhstan company vacancy source discovery — batch 1

Date: 2026-09-25  
Scope: the 20 URLs explicitly authorized for batch 1. This was a read-only, dry-run discovery. No database, ingestion, staging, production, or source registry changes were made.

## Method and limits

Each URL was passed to the existing safe Go website scanner with a per-site limit of 15 outbound HTTP requests, the existing SSRF/public-address validation, and the existing 3 MiB response limit. The sites were scanned sequentially. Total observed HTTP requests: **69 / 300 maximum**. No credentials, cookies, request bodies, or private data were collected.

The static pass ran first for all 20 sites. The browser fallback was **not run** in this discovery batch: the current renderer is an optional production scanner path with a 100-request internal budget, while this research request requires a separate 40-request/site cap. No code was changed solely to run an unbounded diagnostic. Sites with zero static results are therefore recorded as unresolved or blocked; this is not evidence that their public vacancies do not exist.

## Results

| Company | URL | Primary classification | Static requests | Vacancy URLs | Normalized | Skipped | Status / reason |
|---|---|---:|---:|---:|---:|---:|---|
| Halyk Bank | https://halykbank.kz/about/career | STATIC_HTML | 4 | 2 | 2 | 0 | succeeded |
| Tele2 | https://tele2.kz/career | JS_SPA_UNRESOLVED | 2 | 0 | 0 | 0 | succeeded, no static vacancy links |
| Technodom | https://hr.technodom.kz/ | STATIC_HTML | 15 | 39 | 12 | 27 | request budget exhausted |
| Sulpak | https://job.sulpak.kz/WorkMap/WorkMap | JS_SPA_UNRESOLVED | 2 | 0 | 0 | 0 | succeeded, no static vacancy links |
| BI Group | https://bi.group/ru/career | BLOCKED | 1 | 0 | 0 | 0 | HTTP_FAILED |
| Kolesa Group | https://kolesa.group/career/job | STATIC_HTML | 5 | 3 | 3 | 0 | succeeded |
| ERG | https://erg.kz/ru/career | JS_SPA_UNRESOLVED | 2 | 0 | 0 | 0 | succeeded, no static vacancy links |
| Magnum | https://hr.magnum.kz/ | JS_SPA_UNRESOLVED | 3 | 0 | 0 | 0 | succeeded, no static vacancy links |
| Kazakhmys | https://www.kazakhmys.kz/ru/career | JS_SPA_UNRESOLVED | 2 | 0 | 0 | 0 | succeeded, no static vacancy links |
| Qarmet | https://job.qarmet.kz/ | BLOCKED | 1 | 0 | 0 | 0 | TRANSPORT_FAILED |
| QazaqGaz | https://www.qazaqgaz.kz/ru/karera | JS_SPA_UNRESOLVED | 2 | 0 | 0 | 0 | succeeded, no static vacancy links |
| Samruk-Energy | https://samruk-energy.kz/ru/company/vacancy | JS_SPA_UNRESOLVED | 2 | 0 | 0 | 0 | succeeded, no static vacancy links |
| Kazatomprom | https://hrekap.kazatomprom.kz/ | JS_SPA_UNRESOLVED | 3 | 0 | 0 | 0 | succeeded, no static vacancy links |
| KazMunayGas | http://work.kmg.kz/ | JS_SPA_UNRESOLVED | 4 | 0 | 0 | 0 | redirected to HTTPS; no static vacancy links |
| Tengizchevroil | https://tcoext.azure.chevron.com/ru/career/vacancy | BLOCKED | 1 | 0 | 0 | 0 | TRANSPORT_FAILED |
| SMALL | https://small.kz/ru/astana/job | BLOCKED | 1 | 0 | 0 | 0 | HTTP_FAILED |
| KEGOC | https://www.kegoc.kz/ | JS_SPA_UNRESOLVED | 2 | 0 | 0 | 0 | career URL was not exposed in static HTML |
| RG Brands | https://rgbrands.com/career | JS_SPA_UNRESOLVED | 3 | 0 | 0 | 0 | succeeded, no static vacancy links |
| Beeline Kazakhstan | https://beeline.kz/ru | JS_SPA_UNRESOLVED | 13 | 0 | 0 | 0 | succeeded, no static vacancy links |
| Astana Motors Manufacturing Kazakhstan | https://www.astanamotors-manufacturing.kz/ | BLOCKED | 1 | 0 | 0 | 0 | HTTP_FAILED |

Classification is intentionally conservative. `JS_SPA_UNRESOLVED` means the bounded static pass did not expose usable vacancy records; it does not claim that a browser/API endpoint exists. No approved source was detected as a known ATS during this pass.

## Normalized samples

The three successful static sources produced these public vacancy URLs:

- Halyk Bank: [vacancies/2](https://halykbank.kz/about/career/vacancies/2), [vacancies/1](https://halykbank.kz/about/career/vacancies/1). The returned page is a vacancy listing page; the scanner preserved the original URL and left employer, location, salary, and employment type unknown where the page did not provide a field in the normalized detail shape.
- Technodom: [Оператор котельной](https://hr.technodom.kz/vacancies/operator-kotelnoj/), [Оператор ПРТ](https://hr.technodom.kz/vacancies/operator-prt/), [Энергетик](https://hr.technodom.kz/vacancies/energetik/), [Инженер-электрик](https://hr.technodom.kz/vacancies/inzhener-elektrik-2/), [Слесарь-сантехник](https://hr.technodom.kz/vacancies/slesar-santehnik/). Twelve records normalized; 27 candidate links were skipped as malformed or non-vacancy pages within the 15-request budget.
- Kolesa Group: [Middle GO Backend-разработчик Core team](https://kolesa.group/career/job/middle-go-backend-razrabotcik-core-team-136951220), [Fullstack Web QA](https://kolesa.group/career/job/fullstack-web-qa-v-krishakz-136312001), [Разработчик бизнес процессов (Middle)](https://kolesa.group/career/job/razrabotcik-biznes-processov-middle-137563145). Three records normalized.

No normalized rows were persisted. No browser-rendered sample was produced.

## Follow-up boundary

The unresolved set is suitable for a separately approved browser/API diagnostic, capped at five sites and 40 browser/resource requests per site. This document does not authorize that follow-up, an adapter, a source-registry entry, or production enablement. Existing proven Kaspi, Kcell, and Air Astana sources were not rescanned.

