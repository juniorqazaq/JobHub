DROP TABLE IF EXISTS jobhub.company_follows;

ALTER TABLE jobhub.companies
    DROP COLUMN IF EXISTS city,
    DROP COLUMN IF EXISTS industry,
    DROP COLUMN IF EXISTS logo_url,
    DROP COLUMN IF EXISTS website_url,
    DROP COLUMN IF EXISTS description;
