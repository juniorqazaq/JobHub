CREATE TABLE jobhub.job_sources (
    source text PRIMARY KEY,
    provider text NOT NULL,
    market text,
    display_name text NOT NULL,
    enabled boolean NOT NULL DEFAULT true,
    production_permissions_confirmed boolean NOT NULL DEFAULT false,
    attribution_text text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT job_sources_source_not_blank CHECK (length(btrim(source)) > 0),
    CONSTRAINT job_sources_provider_not_blank CHECK (length(btrim(provider)) > 0)
);

INSERT INTO jobhub.job_sources (
    source,
    provider,
    market,
    display_name,
    enabled,
    production_permissions_confirmed,
    attribution_text
) VALUES
    ('jobhub', 'jobhub', 'KZ', 'JobHub', true, true, NULL),
    (
        'jooble:kz',
        'jooble',
        'KZ',
        'Jooble',
        true,
        false,
        'Proof of concept only; production permissions are not confirmed'
    );

CREATE TABLE jobhub.companies (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT companies_name_not_blank CHECK (length(btrim(name)) > 0)
);

CREATE TABLE jobhub.company_sources (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id uuid NOT NULL REFERENCES jobhub.companies(id) ON DELETE CASCADE,
    source text NOT NULL REFERENCES jobhub.job_sources(source) ON DELETE RESTRICT,
    external_id text NOT NULL,
    source_name text,
    source_url text,
    first_seen_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL,
    last_synced_at timestamptz NOT NULL,
    UNIQUE (source, external_id),
    UNIQUE (id, source),
    CONSTRAINT company_sources_external_id_not_blank CHECK (length(btrim(external_id)) > 0),
    CONSTRAINT company_sources_seen_order CHECK (last_seen_at >= first_seen_at)
);

CREATE TABLE jobhub.jobs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source text NOT NULL REFERENCES jobhub.job_sources(source) ON DELETE RESTRICT,
    external_id text,
    source_url text,
    upstream_source_name text,
    company_id uuid REFERENCES jobhub.companies(id) ON DELETE SET NULL,
    company_source_id uuid,
    company_name_raw text,
    title text NOT NULL,
    location_raw text,
    description text,
    description_kind text NOT NULL DEFAULT 'snippet',
    employment_type_raw text,
    salary_raw text,
    salary_min numeric(14,2),
    salary_max numeric(14,2),
    salary_currency text,
    salary_period text,
    salary_gross boolean,
    salary_is_estimated boolean,
    application_method text NOT NULL,
    apply_url text,
    source_status text NOT NULL DEFAULT 'unknown',
    publication_status text,
    first_seen_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL,
    last_synced_at timestamptz NOT NULL,
    fresh_until timestamptz NOT NULL,
    external_created_at timestamptz,
    external_published_at timestamptz,
    external_updated_at timestamptz,
    external_updated_raw text,
    external_expires_at timestamptz,
    external_archived_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, external_id),
    CONSTRAINT jobs_title_not_blank CHECK (length(btrim(title)) > 0),
    CONSTRAINT jobs_description_kind_valid CHECK (description_kind IN ('full', 'snippet')),
    CONSTRAINT jobs_application_method_valid CHECK (application_method IN ('internal', 'external')),
    CONSTRAINT jobs_source_status_valid CHECK (source_status IN ('unknown', 'active', 'archived', 'expired', 'removed')),
    CONSTRAINT jobs_seen_order CHECK (last_seen_at >= first_seen_at),
    CONSTRAINT jobs_freshness_order CHECK (fresh_until >= last_synced_at),
    CONSTRAINT jobs_salary_bounds CHECK (
        (salary_min IS NULL OR salary_min >= 0)
        AND (salary_max IS NULL OR salary_max >= 0)
        AND (salary_min IS NULL OR salary_max IS NULL OR salary_max >= salary_min)
    ),
    CONSTRAINT jobs_source_identity CHECK (
        (source = 'jobhub' AND external_id IS NULL)
        OR
        (
            source <> 'jobhub'
            AND external_id IS NOT NULL
            AND length(btrim(external_id)) > 0
            AND source_url IS NOT NULL
            AND length(btrim(source_url)) > 0
            AND application_method = 'external'
        )
    ),
    CONSTRAINT jobs_company_source_matches_source FOREIGN KEY (company_source_id, source)
        REFERENCES jobhub.company_sources(id, source) ON DELETE SET NULL
);

CREATE TABLE jobhub.ingestion_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source text NOT NULL REFERENCES jobhub.job_sources(source) ON DELETE RESTRICT,
    status text NOT NULL DEFAULT 'running',
    search_count integer NOT NULL DEFAULT 0,
    request_count integer NOT NULL DEFAULT 0,
    fetched_count integer NOT NULL DEFAULT 0,
    normalized_count integer NOT NULL DEFAULT 0,
    inserted_count integer NOT NULL DEFAULT 0,
    updated_count integer NOT NULL DEFAULT 0,
    skipped_count integer NOT NULL DEFAULT 0,
    error_code text,
    error_message text,
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    CONSTRAINT ingestion_runs_status_valid CHECK (status IN ('running', 'succeeded', 'failed')),
    CONSTRAINT ingestion_runs_request_budget CHECK (request_count BETWEEN 0 AND 50),
    CONSTRAINT ingestion_runs_counts_nonnegative CHECK (
        search_count >= 0
        AND fetched_count >= 0
        AND normalized_count >= 0
        AND inserted_count >= 0
        AND updated_count >= 0
        AND skipped_count >= 0
    ),
    CONSTRAINT ingestion_runs_finished_state CHECK (
        (status = 'running' AND finished_at IS NULL)
        OR (status <> 'running' AND finished_at IS NOT NULL)
    )
);

CREATE INDEX jobs_public_feed_idx
    ON jobhub.jobs (COALESCE(external_published_at, first_seen_at) DESC, id DESC)
    WHERE source_status IN ('unknown', 'active');
CREATE INDEX jobs_source_last_seen_idx ON jobhub.jobs (source, last_seen_at DESC);
CREATE INDEX jobs_company_id_idx ON jobhub.jobs (company_id) WHERE company_id IS NOT NULL;
CREATE INDEX jobs_company_source_id_idx ON jobhub.jobs (company_source_id) WHERE company_source_id IS NOT NULL;
CREATE INDEX company_sources_company_id_idx ON jobhub.company_sources (company_id);
CREATE INDEX ingestion_runs_source_started_idx ON jobhub.ingestion_runs (source, started_at DESC);

ALTER TABLE jobhub.job_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobhub.companies ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobhub.company_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobhub.jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobhub.ingestion_runs ENABLE ROW LEVEL SECURITY;

COMMENT ON COLUMN jobhub.job_sources.production_permissions_confirmed IS
    'Must remain false until API-specific storage, redistribution, retention, attribution, and removal permissions are confirmed.';
COMMENT ON TABLE jobhub.ingestion_runs IS
    'Server-side provider import audit metadata. Never store provider credentials here.';
