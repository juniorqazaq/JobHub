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
    canonical_city_id,
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
    sqlc.narg(canonical_city_id),
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
    canonical_city_id = EXCLUDED.canonical_city_id,
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
  AND jobhub.job_is_public(j)
  AND (sqlc.narg(query)::text IS NULL OR j.title ILIKE '%' || sqlc.narg(query) || '%' OR j.company_name_raw ILIKE '%' || sqlc.narg(query) || '%' OR EXISTS (SELECT 1 FROM unnest(j.skills) skill WHERE skill ILIKE '%' || sqlc.narg(query) || '%'))
  AND (sqlc.narg(city)::text IS NULL OR j.canonical_city_id = sqlc.narg(city))
  AND (COALESCE(cardinality(sqlc.arg(work_modes)::text[]), 0) = 0 OR j.work_mode = ANY(sqlc.arg(work_modes)::text[]))
  AND (sqlc.narg(salary_min)::numeric IS NULL OR (j.salary_visible AND j.salary_currency = sqlc.narg(currency) AND j.salary_period = 'month' AND COALESCE(j.salary_max, j.salary_min) >= sqlc.narg(salary_min)))
  AND (sqlc.narg(experience_level)::text IS NULL OR j.experience_level = sqlc.narg(experience_level))
  AND (sqlc.narg(employment_type)::text IS NULL OR j.employment_type = sqlc.narg(employment_type))
  AND (sqlc.narg(posted_after)::timestamptz IS NULL OR COALESCE(j.external_published_at, j.published_at, j.first_seen_at) >= sqlc.narg(posted_after));

-- name: ListPublicJobs :many
SELECT j.*
FROM jobhub.jobs j
JOIN jobhub.job_sources s ON s.source = j.source
WHERE s.enabled
  AND (NOT sqlc.arg(require_production_permissions)::boolean OR s.production_permissions_confirmed)
  AND jobhub.job_is_public(j)
  AND (sqlc.narg(query)::text IS NULL OR j.title ILIKE '%' || sqlc.narg(query) || '%' OR j.company_name_raw ILIKE '%' || sqlc.narg(query) || '%' OR EXISTS (SELECT 1 FROM unnest(j.skills) skill WHERE skill ILIKE '%' || sqlc.narg(query) || '%'))
  AND (sqlc.narg(city)::text IS NULL OR j.canonical_city_id = sqlc.narg(city))
  AND (COALESCE(cardinality(sqlc.arg(work_modes)::text[]), 0) = 0 OR j.work_mode = ANY(sqlc.arg(work_modes)::text[]))
  AND (sqlc.narg(salary_min)::numeric IS NULL OR (j.salary_visible AND j.salary_currency = sqlc.narg(currency) AND j.salary_period = 'month' AND COALESCE(j.salary_max, j.salary_min) >= sqlc.narg(salary_min)))
  AND (sqlc.narg(experience_level)::text IS NULL OR j.experience_level = sqlc.narg(experience_level))
  AND (sqlc.narg(employment_type)::text IS NULL OR j.employment_type = sqlc.narg(employment_type))
  AND (sqlc.narg(posted_after)::timestamptz IS NULL OR COALESCE(j.external_published_at, j.published_at, j.first_seen_at) >= sqlc.narg(posted_after))
ORDER BY
    CASE WHEN sqlc.narg(preferred_city)::text IS NOT NULL AND j.canonical_city_id = sqlc.narg(preferred_city) THEN 0 ELSE 1 END,
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
  AND jobhub.job_is_public(j);

-- name: ListEmployerJobs :many
SELECT j.*
FROM jobhub.jobs j
JOIN jobhub.company_memberships cm ON cm.company_id = j.company_id
WHERE cm.user_id = sqlc.arg(user_id)
  AND cm.role = 'owner'
  AND j.source = 'jobhub'
  AND j.deleted_at IS NULL
ORDER BY j.updated_at DESC, j.id DESC;

-- name: GetEmployerJob :one
SELECT j.*
FROM jobhub.jobs j
JOIN jobhub.company_memberships cm ON cm.company_id = j.company_id
WHERE j.id = sqlc.arg(id)
  AND cm.user_id = sqlc.arg(user_id)
  AND cm.role = 'owner'
  AND j.source = 'jobhub'
  AND j.deleted_at IS NULL;

-- name: CreateNativeJob :one
INSERT INTO jobhub.jobs (
    source, company_id, created_by_user_id, company_name_raw, title, category, description, description_kind,
    responsibilities, requirements, nice_to_have, skills, location_raw, canonical_city_id,
    work_mode, employment_type, experience_level, salary_min, salary_max,
    salary_currency, salary_period, salary_visible, benefits, application_method,
    source_status, publication_status, first_seen_at, last_seen_at, last_synced_at,
    fresh_until, expires_at, moderation_status
)
SELECT
    'jobhub', cm.company_id, sqlc.arg(user_id), c.name, sqlc.arg(title), sqlc.arg(category),
    sqlc.arg(description), 'full', sqlc.arg(responsibilities), sqlc.arg(requirements),
    sqlc.narg(nice_to_have), sqlc.arg(skills)::text[], sqlc.arg(location), sqlc.arg(canonical_city_id),
    sqlc.arg(work_mode), sqlc.arg(employment_type), sqlc.arg(experience_level),
    sqlc.narg(salary_min), sqlc.narg(salary_max), sqlc.narg(salary_currency),
    sqlc.narg(salary_period), sqlc.arg(salary_visible), sqlc.arg(benefits)::text[],
    'internal', 'active', 'draft', now(), now(), now(), 'infinity'::timestamptz,
    sqlc.narg(expires_at), 'approved'
FROM jobhub.company_memberships cm
JOIN jobhub.companies c ON c.id = cm.company_id
JOIN jobhub.users u ON u.id = cm.user_id
WHERE cm.user_id = sqlc.arg(user_id)
  AND cm.role = 'owner'
  AND c.status = 'active'
  AND u.status = 'active'
ORDER BY cm.created_at
LIMIT 1
RETURNING *;

-- name: UpdateNativeJob :one
UPDATE jobhub.jobs j
SET title = sqlc.arg(title),
    category = sqlc.arg(category),
    description = sqlc.arg(description),
    responsibilities = sqlc.arg(responsibilities),
    requirements = sqlc.arg(requirements),
    nice_to_have = sqlc.narg(nice_to_have),
    skills = sqlc.arg(skills)::text[],
    location_raw = sqlc.arg(location),
    canonical_city_id = sqlc.arg(canonical_city_id),
    work_mode = sqlc.arg(work_mode),
    employment_type = sqlc.arg(employment_type),
    experience_level = sqlc.arg(experience_level),
    salary_min = sqlc.narg(salary_min),
    salary_max = sqlc.narg(salary_max),
    salary_currency = sqlc.narg(salary_currency),
    salary_period = sqlc.narg(salary_period),
    salary_visible = sqlc.arg(salary_visible),
    benefits = sqlc.arg(benefits)::text[],
    expires_at = sqlc.narg(expires_at),
    updated_at = now()
WHERE j.id = sqlc.arg(id)
  AND j.source = 'jobhub'
  AND j.deleted_at IS NULL
  AND j.publication_status <> 'closed'
  AND EXISTS (
      SELECT 1 FROM jobhub.company_memberships cm
      WHERE cm.company_id = j.company_id
        AND cm.user_id = sqlc.arg(user_id)
        AND cm.role = 'owner'
  )
RETURNING j.*;

-- name: TransitionNativeJob :one
UPDATE jobhub.jobs j
SET publication_status = sqlc.arg(publication_status),
    published_at = CASE
        WHEN sqlc.arg(publication_status)::text = 'published' THEN COALESCE(j.published_at, now())
        ELSE j.published_at
    END,
    updated_at = now()
WHERE j.id = sqlc.arg(id)
  AND j.source = 'jobhub'
  AND j.deleted_at IS NULL
  AND (
      j.publication_status = sqlc.arg(publication_status)
      OR (j.publication_status = 'draft' AND sqlc.arg(publication_status)::text IN ('published', 'closed'))
      OR (j.publication_status = 'published' AND sqlc.arg(publication_status)::text IN ('paused', 'closed'))
      OR (j.publication_status = 'paused' AND sqlc.arg(publication_status)::text IN ('published', 'closed'))
  )
  AND EXISTS (
      SELECT 1 FROM jobhub.company_memberships cm
      WHERE cm.company_id = j.company_id
        AND cm.user_id = sqlc.arg(user_id)
        AND cm.role = 'owner'
  )
RETURNING j.*;

-- name: SoftDeleteNativeJob :one
UPDATE jobhub.jobs j
SET deleted_at = now(), updated_at = now()
WHERE j.id = sqlc.arg(id)
  AND j.source = 'jobhub'
  AND j.deleted_at IS NULL
  AND j.publication_status IN ('draft', 'closed')
  AND EXISTS (
      SELECT 1 FROM jobhub.company_memberships cm
      WHERE cm.company_id = j.company_id
        AND cm.user_id = sqlc.arg(user_id)
        AND cm.role = 'owner'
  )
RETURNING j.id;
