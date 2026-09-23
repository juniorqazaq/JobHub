-- name: CreateIngestionRun :one
INSERT INTO jobhub.ingestion_runs (source, search_count)
VALUES (sqlc.arg(source), sqlc.arg(search_count))
RETURNING id;

-- name: FailIngestionRun :exec
UPDATE jobhub.ingestion_runs
SET status = 'failed',
    request_count = sqlc.arg(request_count),
    fetched_count = sqlc.arg(fetched_count),
    normalized_count = sqlc.arg(normalized_count),
    error_code = sqlc.arg(error_code),
    error_message = sqlc.arg(error_message),
    finished_at = now()
WHERE id = sqlc.arg(id) AND status = 'running';

-- name: CompleteIngestionRun :exec
UPDATE jobhub.ingestion_runs
SET status = 'succeeded',
    request_count = sqlc.arg(request_count),
    fetched_count = sqlc.arg(fetched_count),
    normalized_count = sqlc.arg(normalized_count),
    inserted_count = sqlc.arg(inserted_count),
    updated_count = sqlc.arg(updated_count),
    skipped_count = sqlc.arg(skipped_count),
    finished_at = now()
WHERE id = sqlc.arg(id) AND status = 'running';

-- name: AcquireSourceIngestionLock :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(source), 0));

-- name: ImportedJobExists :one
SELECT EXISTS (
    SELECT 1 FROM jobhub.jobs
    WHERE source = sqlc.arg(source) AND external_id = sqlc.arg(external_id)
);

-- name: UpsertImportedJob :one
INSERT INTO jobhub.jobs (
    source,
    external_id,
    source_url,
    upstream_source_name,
    company_name_raw,
    title,
    location_raw,
    description,
    description_kind,
    employment_type_raw,
    salary_raw,
    application_method,
    apply_url,
    source_status,
    first_seen_at,
    last_seen_at,
    last_synced_at,
    fresh_until,
    external_updated_at,
    external_updated_raw
) VALUES (
    sqlc.arg(source),
    sqlc.arg(external_id),
    sqlc.arg(source_url),
    sqlc.narg(upstream_source_name),
    sqlc.narg(company_name_raw),
    sqlc.arg(title),
    sqlc.narg(location_raw),
    sqlc.narg(description),
    sqlc.arg(description_kind),
    sqlc.narg(employment_type_raw),
    sqlc.narg(salary_raw),
    'external',
    sqlc.arg(source_url),
    'unknown',
    sqlc.arg(observed_at),
    sqlc.arg(observed_at),
    sqlc.arg(observed_at),
    sqlc.arg(fresh_until),
    sqlc.narg(external_updated_at),
    sqlc.narg(external_updated_raw)
)
ON CONFLICT (source, external_id) DO UPDATE SET
    source_url = EXCLUDED.source_url,
    upstream_source_name = EXCLUDED.upstream_source_name,
    company_name_raw = EXCLUDED.company_name_raw,
    title = EXCLUDED.title,
    location_raw = EXCLUDED.location_raw,
    description = EXCLUDED.description,
    description_kind = EXCLUDED.description_kind,
    employment_type_raw = EXCLUDED.employment_type_raw,
    salary_raw = EXCLUDED.salary_raw,
    apply_url = EXCLUDED.apply_url,
    last_seen_at = EXCLUDED.last_seen_at,
    last_synced_at = EXCLUDED.last_synced_at,
    fresh_until = EXCLUDED.fresh_until,
    external_updated_at = EXCLUDED.external_updated_at,
    external_updated_raw = EXCLUDED.external_updated_raw,
    updated_at = now()
WHERE jobhub.jobs.last_synced_at <= EXCLUDED.last_synced_at
RETURNING id;

-- name: CountPublicJobs :one
SELECT count(*)
FROM jobhub.jobs j
JOIN jobhub.job_sources s ON s.source = j.source
WHERE s.enabled
  AND (NOT sqlc.arg(require_production_permissions)::boolean OR s.production_permissions_confirmed)
  AND j.source_status IN ('unknown', 'active')
  AND j.fresh_until > now()
  AND (sqlc.narg(query)::text IS NULL OR j.title ILIKE '%' || sqlc.narg(query) || '%' OR j.company_name_raw ILIKE '%' || sqlc.narg(query) || '%')
  AND (sqlc.narg(location)::text IS NULL OR j.location_raw ILIKE '%' || sqlc.narg(location) || '%');

-- name: ListPublicJobs :many
SELECT j.*
FROM jobhub.jobs j
JOIN jobhub.job_sources s ON s.source = j.source
WHERE s.enabled
  AND (NOT sqlc.arg(require_production_permissions)::boolean OR s.production_permissions_confirmed)
  AND j.source_status IN ('unknown', 'active')
  AND j.fresh_until > now()
  AND (sqlc.narg(query)::text IS NULL OR j.title ILIKE '%' || sqlc.narg(query) || '%' OR j.company_name_raw ILIKE '%' || sqlc.narg(query) || '%')
  AND (sqlc.narg(location)::text IS NULL OR j.location_raw ILIKE '%' || sqlc.narg(location) || '%')
ORDER BY
    CASE WHEN sqlc.arg(sort)::text = 'oldest' THEN COALESCE(j.external_published_at, j.first_seen_at) END ASC,
    CASE WHEN sqlc.arg(sort)::text <> 'oldest' THEN COALESCE(j.external_published_at, j.first_seen_at) END DESC,
    j.id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: GetPublicJob :one
SELECT j.*
FROM jobhub.jobs j
JOIN jobhub.job_sources s ON s.source = j.source
WHERE j.id = sqlc.arg(id)
  AND s.enabled
  AND (NOT sqlc.arg(require_production_permissions)::boolean OR s.production_permissions_confirmed)
  AND j.source_status IN ('unknown', 'active')
  AND j.fresh_until > now();
