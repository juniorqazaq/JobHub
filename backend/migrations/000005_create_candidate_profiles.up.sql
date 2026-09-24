CREATE TABLE jobhub.candidate_profiles (
    user_id uuid PRIMARY KEY REFERENCES jobhub.users(id) ON DELETE CASCADE,
    photo_url text,
    city text,
    birth_year integer,
    phone text,
    about text,
    current_position text,
    desired_position text,
    years_experience numeric(4,1),
    experience_level text,
    certifications text[] NOT NULL DEFAULT '{}',
    desired_salary numeric(14,2),
    currency text,
    salary_period text,
    preferred_locations text[] NOT NULL DEFAULT '{}',
    preferred_employment_types text[] NOT NULL DEFAULT '{}',
    preferred_work_modes text[] NOT NULL DEFAULT '{}',
    preferred_categories text[] NOT NULL DEFAULT '{}',
    preferred_roles text[] NOT NULL DEFAULT '{}',
    search_status text NOT NULL DEFAULT 'actively_looking',
    github_url text,
    linkedin_url text,
    portfolio_url text,
    website_url text,
    allow_employer_contact boolean NOT NULL DEFAULT false,
    show_profile_to_employers boolean NOT NULL DEFAULT false,
    show_salary_expectations boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT candidate_profiles_birth_year_valid CHECK (birth_year IS NULL OR birth_year BETWEEN 1900 AND 2100),
    CONSTRAINT candidate_profiles_experience_years_valid CHECK (years_experience IS NULL OR years_experience BETWEEN 0 AND 80),
    CONSTRAINT candidate_profiles_experience_level_valid CHECK (
        experience_level IS NULL OR experience_level IN ('internship', 'junior', 'middle', 'senior', 'lead')
    ),
    CONSTRAINT candidate_profiles_salary_valid CHECK (desired_salary IS NULL OR desired_salary >= 0),
    CONSTRAINT candidate_profiles_salary_period_valid CHECK (salary_period IS NULL OR salary_period IN ('month', 'year')),
    CONSTRAINT candidate_profiles_search_status_valid CHECK (search_status IN ('actively_looking', 'open_to_offers', 'not_looking'))
);

CREATE TABLE jobhub.work_experience (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id uuid NOT NULL REFERENCES jobhub.candidate_profiles(user_id) ON DELETE CASCADE,
    company text NOT NULL,
    position text NOT NULL,
    employment_type text NOT NULL,
    start_date date NOT NULL,
    end_date date,
    is_current boolean NOT NULL DEFAULT false,
    description text,
    achievements text,
    skills text[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT work_experience_company_not_blank CHECK (length(btrim(company)) > 0),
    CONSTRAINT work_experience_position_not_blank CHECK (length(btrim(position)) > 0),
    CONSTRAINT work_experience_employment_type_valid CHECK (
        employment_type IN ('full_time', 'part_time', 'contract', 'temporary', 'internship')
    ),
    CONSTRAINT work_experience_dates_valid CHECK (
        (is_current AND end_date IS NULL) OR (NOT is_current AND end_date IS NOT NULL AND end_date >= start_date)
    )
);

CREATE TABLE jobhub.education (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id uuid NOT NULL REFERENCES jobhub.candidate_profiles(user_id) ON DELETE CASCADE,
    institution text NOT NULL,
    degree text NOT NULL,
    field_of_study text NOT NULL,
    start_year integer NOT NULL,
    graduation_year integer,
    description text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT education_institution_not_blank CHECK (length(btrim(institution)) > 0),
    CONSTRAINT education_years_valid CHECK (
        start_year BETWEEN 1900 AND 2100
        AND (graduation_year IS NULL OR (graduation_year BETWEEN start_year AND 2100))
    )
);

CREATE TABLE jobhub.skills (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    normalized_name text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT skills_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT skills_normalized_name_not_blank CHECK (length(btrim(normalized_name)) > 0)
);

CREATE TABLE jobhub.candidate_skills (
    candidate_id uuid NOT NULL REFERENCES jobhub.candidate_profiles(user_id) ON DELETE CASCADE,
    skill_id uuid NOT NULL REFERENCES jobhub.skills(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (candidate_id, skill_id)
);

CREATE TABLE jobhub.candidate_languages (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id uuid NOT NULL REFERENCES jobhub.candidate_profiles(user_id) ON DELETE CASCADE,
    language text NOT NULL,
    normalized_language text NOT NULL,
    proficiency text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (candidate_id, normalized_language),
    CONSTRAINT candidate_languages_name_not_blank CHECK (length(btrim(language)) > 0),
    CONSTRAINT candidate_languages_proficiency_valid CHECK (proficiency IN ('native', 'fluent', 'A1', 'A2', 'B1', 'B2', 'C1', 'C2'))
);

CREATE INDEX work_experience_candidate_idx ON jobhub.work_experience (candidate_id, start_date DESC);
CREATE INDEX education_candidate_idx ON jobhub.education (candidate_id, start_year DESC);
CREATE INDEX candidate_languages_candidate_idx ON jobhub.candidate_languages (candidate_id);

ALTER TABLE jobhub.candidate_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobhub.work_experience ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobhub.education ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobhub.skills ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobhub.candidate_skills ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobhub.candidate_languages ENABLE ROW LEVEL SECURITY;

COMMENT ON TABLE jobhub.candidate_profiles IS 'Private professional profiles accessed only through the authorized Go API.';
