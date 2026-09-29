ALTER TABLE jobhub.companies
    ADD COLUMN description text,
    ADD COLUMN website_url text,
    ADD COLUMN logo_url text,
    ADD COLUMN industry text,
    ADD COLUMN city text;

CREATE TABLE jobhub.company_follows (
    candidate_id uuid NOT NULL REFERENCES jobhub.users(id) ON DELETE CASCADE,
    company_id uuid NOT NULL REFERENCES jobhub.companies(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (candidate_id, company_id)
);

CREATE INDEX company_follows_company_created_idx
    ON jobhub.company_follows (company_id, created_at DESC);

ALTER TABLE jobhub.company_follows ENABLE ROW LEVEL SECURITY;

COMMENT ON COLUMN jobhub.companies.logo_url IS
    'HTTPS or application-local URL used for the public company profile logo.';
