CREATE INDEX IF NOT EXISTS jobs_company_public_published_idx
    ON jobhub.jobs (
        company_id,
        COALESCE(published_at, external_published_at, first_seen_at) DESC,
        id DESC
    )
    WHERE company_id IS NOT NULL
      AND deleted_at IS NULL
      AND moderation_status = 'approved';

CREATE INDEX IF NOT EXISTS jobs_source_public_published_idx
    ON jobhub.jobs (
        source,
        COALESCE(published_at, external_published_at, first_seen_at) DESC,
        id DESC
    )
    WHERE deleted_at IS NULL
      AND moderation_status = 'approved';
