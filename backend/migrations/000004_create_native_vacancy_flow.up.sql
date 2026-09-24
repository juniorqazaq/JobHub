ALTER TABLE jobhub.jobs
    ADD COLUMN category text,
    ADD COLUMN responsibilities text,
    ADD COLUMN requirements text,
    ADD COLUMN nice_to_have text,
    ADD COLUMN skills text[] NOT NULL DEFAULT '{}',
    ADD COLUMN work_mode text,
    ADD COLUMN employment_type text,
    ADD COLUMN experience_level text,
    ADD COLUMN benefits text[] NOT NULL DEFAULT '{}',
    ADD COLUMN salary_visible boolean NOT NULL DEFAULT true,
    ADD COLUMN expires_at timestamptz,
    ADD COLUMN published_at timestamptz,
    ADD COLUMN deleted_at timestamptz,
    ADD COLUMN moderation_status text NOT NULL DEFAULT 'approved',
    ADD COLUMN created_by_user_id uuid REFERENCES jobhub.users(id) ON DELETE RESTRICT,
    ADD CONSTRAINT jobs_publication_status_valid CHECK (
        publication_status IS NULL OR publication_status IN ('draft', 'published', 'paused', 'closed')
    ),
    ADD CONSTRAINT jobs_work_mode_valid CHECK (
        work_mode IS NULL OR work_mode IN ('on_site', 'hybrid', 'remote')
    ),
    ADD CONSTRAINT jobs_employment_type_valid CHECK (
        employment_type IS NULL OR employment_type IN ('full_time', 'part_time', 'contract', 'temporary', 'internship')
    ),
    ADD CONSTRAINT jobs_experience_level_valid CHECK (
        experience_level IS NULL OR experience_level IN ('no_experience', 'junior', 'middle', 'senior', 'lead')
    ),
    ADD CONSTRAINT jobs_moderation_status_valid CHECK (
        moderation_status IN ('approved', 'pending', 'rejected')
    ),
    ADD CONSTRAINT jobs_native_payload_valid CHECK (
        source <> 'jobhub'
        OR (
            company_id IS NOT NULL
            AND created_by_user_id IS NOT NULL
            AND publication_status IS NOT NULL
            AND application_method = 'internal'
            AND external_id IS NULL
            AND source_url IS NULL
            AND apply_url IS NULL
        )
    ),
    ADD CONSTRAINT jobs_expiry_valid CHECK (expires_at IS NULL OR expires_at > created_at);

CREATE INDEX jobs_employer_list_idx
    ON jobhub.jobs (company_id, updated_at DESC)
    WHERE source = 'jobhub' AND deleted_at IS NULL;

CREATE INDEX jobs_native_public_idx
    ON jobhub.jobs (published_at DESC, id DESC)
    WHERE source = 'jobhub'
      AND publication_status = 'published'
      AND deleted_at IS NULL
      AND moderation_status = 'approved';

CREATE OR REPLACE FUNCTION jobhub.job_is_public(candidate jobhub.jobs)
RETURNS boolean
LANGUAGE sql
STABLE
AS $$
    SELECT candidate.deleted_at IS NULL
       AND candidate.moderation_status = 'approved'
       AND (
           (
               candidate.source = 'jobhub'
               AND candidate.publication_status = 'published'
               AND (candidate.expires_at IS NULL OR candidate.expires_at > now())
               AND EXISTS (
                   SELECT 1
                   FROM jobhub.companies c
                   WHERE c.id = candidate.company_id
                     AND c.status = 'active'
               )
               AND EXISTS (
                   SELECT 1
                   FROM jobhub.company_memberships cm
                   JOIN jobhub.users u ON u.id = cm.user_id
                   WHERE cm.company_id = candidate.company_id
                     AND cm.role = 'owner'
                     AND u.status = 'active'
               )
           )
           OR
           (
               candidate.source <> 'jobhub'
               AND candidate.source_status IN ('unknown', 'active')
               AND candidate.fresh_until > now()
           )
       );
$$;

COMMENT ON FUNCTION jobhub.job_is_public(jobhub.jobs) IS
    'Shared public visibility predicate for native and imported vacancies.';
