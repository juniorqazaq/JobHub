CREATE TABLE jobhub.applications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id uuid NOT NULL REFERENCES jobhub.jobs(id) ON DELETE RESTRICT,
    company_id uuid NOT NULL REFERENCES jobhub.companies(id) ON DELETE RESTRICT,
    candidate_id uuid NOT NULL REFERENCES jobhub.users(id) ON DELETE RESTRICT,
    resume_id uuid NOT NULL REFERENCES jobhub.resumes(id) ON DELETE RESTRICT,
    status text NOT NULL DEFAULT 'sent',
    message text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (candidate_id, job_id),
    CONSTRAINT applications_status_valid CHECK (status IN ('sent', 'viewed', 'in_review', 'contacted', 'interview', 'offer', 'rejected', 'withdrawn')),
    CONSTRAINT applications_message_length CHECK (message IS NULL OR length(message) <= 3000)
);

CREATE TABLE jobhub.application_status_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id uuid NOT NULL REFERENCES jobhub.applications(id) ON DELETE CASCADE,
    actor_user_id uuid NOT NULL REFERENCES jobhub.users(id) ON DELETE RESTRICT,
    previous_status text,
    new_status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT application_events_previous_status_valid CHECK (
        previous_status IS NULL OR previous_status IN ('sent', 'viewed', 'in_review', 'contacted', 'interview', 'offer', 'rejected', 'withdrawn')
    ),
    CONSTRAINT application_events_new_status_valid CHECK (
        new_status IN ('sent', 'viewed', 'in_review', 'contacted', 'interview', 'offer', 'rejected', 'withdrawn')
    )
);

CREATE INDEX applications_candidate_created_idx ON jobhub.applications (candidate_id, created_at DESC);
CREATE INDEX applications_company_job_created_idx ON jobhub.applications (company_id, job_id, created_at DESC);
CREATE INDEX application_status_events_application_idx ON jobhub.application_status_events (application_id, created_at);

ALTER TABLE jobhub.applications ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobhub.application_status_events ENABLE ROW LEVEL SECURITY;

COMMENT ON COLUMN jobhub.applications.resume_id IS 'Immutable submitted resume version. Active profile resume changes do not alter historical applications.';
