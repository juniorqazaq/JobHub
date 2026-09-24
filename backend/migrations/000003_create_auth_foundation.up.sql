CREATE TABLE jobhub.users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name text NOT NULL,
    email text NOT NULL,
    normalized_email text NOT NULL,
    password_hash text NOT NULL,
    role text NOT NULL,
    status text NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (normalized_email),
    CONSTRAINT users_full_name_not_blank CHECK (length(btrim(full_name)) > 0),
    CONSTRAINT users_email_not_blank CHECK (length(btrim(email)) > 0),
    CONSTRAINT users_normalized_email_not_blank CHECK (length(btrim(normalized_email)) > 0),
    CONSTRAINT users_role_valid CHECK (role IN ('job_seeker', 'employer', 'admin')),
    CONSTRAINT users_status_valid CHECK (status IN ('active', 'suspended'))
);

ALTER TABLE jobhub.companies
    ADD COLUMN IF NOT EXISTS status text NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS is_verified boolean NOT NULL DEFAULT false,
    ADD CONSTRAINT companies_status_valid CHECK (status IN ('active', 'inactive', 'suspended'));

CREATE TABLE jobhub.company_memberships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id uuid NOT NULL REFERENCES jobhub.companies(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES jobhub.users(id) ON DELETE CASCADE,
    role text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (company_id, user_id),
    CONSTRAINT company_memberships_role_valid CHECK (role IN ('owner'))
);

CREATE UNIQUE INDEX company_memberships_one_owner_idx
    ON jobhub.company_memberships (company_id)
    WHERE role = 'owner';
CREATE INDEX company_memberships_user_id_idx ON jobhub.company_memberships (user_id);

CREATE TABLE jobhub.sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES jobhub.users(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    csrf_hash bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sessions_expiry_valid CHECK (expires_at > created_at)
);

CREATE INDEX sessions_user_id_idx ON jobhub.sessions (user_id);
CREATE INDEX sessions_active_lookup_idx ON jobhub.sessions (token_hash, expires_at) WHERE revoked_at IS NULL;

ALTER TABLE jobhub.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobhub.company_memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobhub.sessions ENABLE ROW LEVEL SECURITY;

COMMENT ON TABLE jobhub.sessions IS
    'Stores only hashed opaque session and CSRF tokens. Raw tokens are never persisted.';
