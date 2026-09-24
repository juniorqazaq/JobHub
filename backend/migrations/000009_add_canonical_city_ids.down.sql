DROP INDEX IF EXISTS jobhub.jobs_canonical_city_public_idx;
ALTER TABLE jobhub.candidate_profiles
    DROP CONSTRAINT IF EXISTS candidate_preferred_cities_valid,
    DROP CONSTRAINT IF EXISTS candidate_city_valid,
    DROP COLUMN IF EXISTS preferred_city_ids,
    DROP COLUMN IF EXISTS canonical_city_id;
ALTER TABLE jobhub.jobs
    DROP CONSTRAINT IF EXISTS jobs_canonical_city_valid,
    DROP COLUMN IF EXISTS canonical_city_id;
