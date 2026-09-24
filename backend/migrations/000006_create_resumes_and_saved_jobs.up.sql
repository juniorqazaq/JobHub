CREATE TABLE jobhub.resumes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id uuid NOT NULL REFERENCES jobhub.users(id) ON DELETE CASCADE,
    storage_key text NOT NULL UNIQUE,
    original_filename text NOT NULL,
    content_type text NOT NULL,
    size_bytes bigint NOT NULL,
    status text NOT NULL DEFAULT 'ready',
    uploaded_at timestamptz NOT NULL DEFAULT now(),
    retired_at timestamptz,
    deleted_at timestamptz,
    CONSTRAINT resumes_storage_key_not_blank CHECK (length(btrim(storage_key)) > 0),
    CONSTRAINT resumes_original_filename_not_blank CHECK (length(btrim(original_filename)) > 0),
    CONSTRAINT resumes_content_type_pdf CHECK (content_type = 'application/pdf'),
    CONSTRAINT resumes_size_valid CHECK (size_bytes BETWEEN 1 AND 10485760),
    CONSTRAINT resumes_status_valid CHECK (status IN ('ready', 'retired', 'deleted'))
);

CREATE UNIQUE INDEX resumes_one_active_candidate_idx
    ON jobhub.resumes (candidate_id)
    WHERE status = 'ready';
CREATE INDEX resumes_candidate_history_idx ON jobhub.resumes (candidate_id, uploaded_at DESC);

CREATE TABLE jobhub.saved_jobs (
    candidate_id uuid NOT NULL REFERENCES jobhub.users(id) ON DELETE CASCADE,
    job_id uuid NOT NULL REFERENCES jobhub.jobs(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (candidate_id, job_id)
);

CREATE INDEX saved_jobs_candidate_created_idx ON jobhub.saved_jobs (candidate_id, created_at DESC);

ALTER TABLE jobhub.resumes ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobhub.saved_jobs ENABLE ROW LEVEL SECURITY;

COMMENT ON COLUMN jobhub.resumes.storage_key IS 'Opaque server-side storage key. Never expose it in API responses.';
