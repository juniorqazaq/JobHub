DROP TABLE IF EXISTS jobhub.sessions;
DROP TABLE IF EXISTS jobhub.company_memberships;
ALTER TABLE jobhub.companies
    DROP CONSTRAINT IF EXISTS companies_status_valid,
    DROP COLUMN IF EXISTS is_verified,
    DROP COLUMN IF EXISTS status;
DROP TABLE IF EXISTS jobhub.users;
