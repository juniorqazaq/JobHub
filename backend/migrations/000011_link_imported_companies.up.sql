-- Imported providers often supply only a display name for the employer. Keep one
-- public company record for each whitespace/case-insensitive name so vacancies
-- from repeated imports resolve to the same profile. Older imports may already
-- contain duplicate company records, so move each relationship to the oldest
-- profile before enforcing the uniqueness invariant.
CREATE TEMP TABLE company_duplicate_map ON COMMIT DROP AS
WITH ranked AS (
    SELECT
        id,
        first_value(id) OVER (
            PARTITION BY lower(regexp_replace(btrim(name), '\s+', ' ', 'g'))
            ORDER BY created_at ASC, id ASC
        ) AS canonical_id
    FROM jobhub.companies
)
SELECT id AS duplicate_id, canonical_id
FROM ranked
WHERE id <> canonical_id;

UPDATE jobhub.jobs j
SET company_id = m.canonical_id,
    updated_at = now()
FROM company_duplicate_map m
WHERE j.company_id = m.duplicate_id;

UPDATE jobhub.company_sources cs
SET company_id = m.canonical_id
FROM company_duplicate_map m
WHERE cs.company_id = m.duplicate_id;

UPDATE jobhub.applications a
SET company_id = m.canonical_id,
    updated_at = now()
FROM company_duplicate_map m
WHERE a.company_id = m.duplicate_id;

-- There can be only one owner per company. Keep the canonical profile's owner
-- when present; otherwise preserve the oldest duplicate owner.
INSERT INTO jobhub.company_memberships (company_id, user_id, role, created_at, updated_at)
SELECT DISTINCT ON (m.canonical_id)
    m.canonical_id,
    cm.user_id,
    cm.role,
    cm.created_at,
    cm.updated_at
FROM company_duplicate_map m
JOIN jobhub.company_memberships cm ON cm.company_id = m.duplicate_id
WHERE NOT EXISTS (
    SELECT 1
    FROM jobhub.company_memberships existing
    WHERE existing.company_id = m.canonical_id
)
ORDER BY m.canonical_id, cm.created_at ASC, cm.id ASC
ON CONFLICT (company_id, user_id) DO NOTHING;

DELETE FROM jobhub.company_memberships cm
USING company_duplicate_map m
WHERE cm.company_id = m.duplicate_id;

INSERT INTO jobhub.company_follows (candidate_id, company_id, created_at)
SELECT cf.candidate_id, m.canonical_id, cf.created_at
FROM company_duplicate_map m
JOIN jobhub.company_follows cf ON cf.company_id = m.duplicate_id
ON CONFLICT (candidate_id, company_id) DO NOTHING;

DELETE FROM jobhub.company_follows cf
USING company_duplicate_map m
WHERE cf.company_id = m.duplicate_id;

DELETE FROM jobhub.companies c
USING company_duplicate_map m
WHERE c.id = m.duplicate_id;

CREATE UNIQUE INDEX companies_normalized_name_key
    ON jobhub.companies (lower(regexp_replace(btrim(name), '\s+', ' ', 'g')));

WITH imported_names AS (
    SELECT DISTINCT btrim(regexp_replace(company_name_raw, '\s+', ' ', 'g')) AS name
    FROM jobhub.jobs
    WHERE source <> 'jobhub'
      AND company_id IS NULL
      AND NULLIF(btrim(company_name_raw), '') IS NOT NULL
), inserted_companies AS (
    INSERT INTO jobhub.companies (name)
    SELECT name FROM imported_names
    ON CONFLICT (lower(regexp_replace(btrim(name), '\s+', ' ', 'g'))) DO NOTHING
    RETURNING id
)
UPDATE jobhub.jobs j
SET company_id = c.id,
    updated_at = now()
FROM jobhub.companies c
WHERE j.source <> 'jobhub'
  AND j.company_id IS NULL
  AND NULLIF(btrim(j.company_name_raw), '') IS NOT NULL
  AND lower(regexp_replace(btrim(j.company_name_raw), '\s+', ' ', 'g')) = lower(regexp_replace(btrim(c.name), '\s+', ' ', 'g'));
