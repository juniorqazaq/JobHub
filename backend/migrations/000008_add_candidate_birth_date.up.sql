ALTER TABLE jobhub.candidate_profiles
    ADD COLUMN birth_date date;

COMMENT ON COLUMN jobhub.candidate_profiles.birth_date IS
    'Optional private full birth date. Legacy birth_year remains unchanged until the candidate explicitly supplies a date.';
