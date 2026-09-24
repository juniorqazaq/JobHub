DROP FUNCTION IF EXISTS jobhub.job_is_public(jobhub.jobs);
DROP INDEX IF EXISTS jobhub.jobs_native_public_idx;
DROP INDEX IF EXISTS jobhub.jobs_employer_list_idx;

ALTER TABLE jobhub.jobs
    DROP CONSTRAINT IF EXISTS jobs_expiry_valid,
    DROP CONSTRAINT IF EXISTS jobs_native_payload_valid,
    DROP CONSTRAINT IF EXISTS jobs_moderation_status_valid,
    DROP CONSTRAINT IF EXISTS jobs_experience_level_valid,
    DROP CONSTRAINT IF EXISTS jobs_employment_type_valid,
    DROP CONSTRAINT IF EXISTS jobs_work_mode_valid,
    DROP CONSTRAINT IF EXISTS jobs_publication_status_valid,
    DROP COLUMN IF EXISTS created_by_user_id,
    DROP COLUMN IF EXISTS moderation_status,
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS published_at,
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS salary_visible,
    DROP COLUMN IF EXISTS benefits,
    DROP COLUMN IF EXISTS experience_level,
    DROP COLUMN IF EXISTS employment_type,
    DROP COLUMN IF EXISTS work_mode,
    DROP COLUMN IF EXISTS skills,
    DROP COLUMN IF EXISTS nice_to_have,
    DROP COLUMN IF EXISTS requirements,
    DROP COLUMN IF EXISTS responsibilities,
    DROP COLUMN IF EXISTS category;
