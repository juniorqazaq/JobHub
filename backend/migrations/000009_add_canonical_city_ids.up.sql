ALTER TABLE jobhub.jobs ADD COLUMN canonical_city_id text;
ALTER TABLE jobhub.candidate_profiles
    ADD COLUMN canonical_city_id text,
    ADD COLUMN preferred_city_ids text[] NOT NULL DEFAULT '{}';

ALTER TABLE jobhub.jobs ADD CONSTRAINT jobs_canonical_city_valid CHECK (
    canonical_city_id IS NULL OR canonical_city_id = ANY (ARRAY[
        'astana','almaty','shymkent','karaganda','aktobe','taraz','pavlodar','oskemen',
        'semey','atyrau','kostanay','kyzylorda','aktau','oral','petropavl','turkistan'
    ]::text[])
);
ALTER TABLE jobhub.candidate_profiles ADD CONSTRAINT candidate_city_valid CHECK (
    canonical_city_id IS NULL OR canonical_city_id = ANY (ARRAY[
        'astana','almaty','shymkent','karaganda','aktobe','taraz','pavlodar','oskemen',
        'semey','atyrau','kostanay','kyzylorda','aktau','oral','petropavl','turkistan'
    ]::text[])
), ADD CONSTRAINT candidate_preferred_cities_valid CHECK (
    preferred_city_ids <@ ARRAY[
        'astana','almaty','shymkent','karaganda','aktobe','taraz','pavlodar','oskemen',
        'semey','atyrau','kostanay','kyzylorda','aktau','oral','petropavl','turkistan'
    ]::text[]
);

CREATE OR REPLACE FUNCTION jobhub.canonical_city_from_legacy(value text)
RETURNS text LANGUAGE sql IMMUTABLE AS $$
    SELECT CASE regexp_replace(
        translate(
            btrim(COALESCE(value, '')),
            'ABCDEFGHIJKLMNOPQRSTUVWXYZАБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯӘҒҚҢӨҰҮҺІ',
            'abcdefghijklmnopqrstuvwxyzабвгдеёжзийклмнопрстуфхцчшщъыьэюяәғқңөұүһі'
        ),
        '[[:space:]]*,[[:space:]]*(kazakhstan|қазақстан|казахстан)[[:space:]]*$',
        '',
        'i'
    )
        WHEN 'astana' THEN 'astana' WHEN 'астана' THEN 'astana' WHEN 'нур-султан' THEN 'astana' WHEN 'нұр-сұлтан' THEN 'astana'
        WHEN 'almaty' THEN 'almaty' WHEN 'алматы' THEN 'almaty'
        WHEN 'shymkent' THEN 'shymkent' WHEN 'шымкент' THEN 'shymkent'
        WHEN 'karaganda' THEN 'karaganda' WHEN 'караганда' THEN 'karaganda' WHEN 'қарағанды' THEN 'karaganda'
        WHEN 'aktobe' THEN 'aktobe' WHEN 'актобе' THEN 'aktobe' WHEN 'ақтөбе' THEN 'aktobe'
        WHEN 'taraz' THEN 'taraz' WHEN 'тараз' THEN 'taraz'
        WHEN 'pavlodar' THEN 'pavlodar' WHEN 'павлодар' THEN 'pavlodar'
        WHEN 'oskemen' THEN 'oskemen' WHEN 'өскемен' THEN 'oskemen' WHEN 'усть-каменогорск' THEN 'oskemen' WHEN 'ust-kamenogorsk' THEN 'oskemen'
        WHEN 'semey' THEN 'semey' WHEN 'семей' THEN 'semey'
        WHEN 'atyrau' THEN 'atyrau' WHEN 'атырау' THEN 'atyrau'
        WHEN 'kostanay' THEN 'kostanay' WHEN 'костанай' THEN 'kostanay' WHEN 'қостанай' THEN 'kostanay'
        WHEN 'kyzylorda' THEN 'kyzylorda' WHEN 'кызылорда' THEN 'kyzylorda' WHEN 'қызылорда' THEN 'kyzylorda'
        WHEN 'aktau' THEN 'aktau' WHEN 'актау' THEN 'aktau' WHEN 'ақтау' THEN 'aktau'
        WHEN 'oral' THEN 'oral' WHEN 'uralsk' THEN 'oral' WHEN 'уральск' THEN 'oral' WHEN 'орал' THEN 'oral'
        WHEN 'petropavl' THEN 'petropavl' WHEN 'petropavlovsk' THEN 'petropavl' WHEN 'петропавл' THEN 'petropavl' WHEN 'петропавловск' THEN 'petropavl'
        WHEN 'turkistan' THEN 'turkistan' WHEN 'туркестан' THEN 'turkistan' WHEN 'түркістан' THEN 'turkistan'
        ELSE NULL END
$$;

UPDATE jobhub.jobs SET canonical_city_id = jobhub.canonical_city_from_legacy(location_raw)
WHERE canonical_city_id IS NULL;
UPDATE jobhub.candidate_profiles SET canonical_city_id = jobhub.canonical_city_from_legacy(city)
WHERE canonical_city_id IS NULL;
UPDATE jobhub.candidate_profiles p SET preferred_city_ids = COALESCE((
    SELECT array_agg(DISTINCT city_id ORDER BY city_id)
    FROM unnest(p.preferred_locations) value
    CROSS JOIN LATERAL (SELECT jobhub.canonical_city_from_legacy(value) city_id) mapped
    WHERE city_id IS NOT NULL
), '{}');

DO $$
DECLARE
    unmapped_jobs bigint;
    unmapped_profile_cities bigint;
    unmapped_preferred_locations bigint;
BEGIN
    SELECT count(*) INTO unmapped_jobs
    FROM jobhub.jobs
    WHERE NULLIF(btrim(location_raw), '') IS NOT NULL AND canonical_city_id IS NULL;

    SELECT count(*) INTO unmapped_profile_cities
    FROM jobhub.candidate_profiles
    WHERE NULLIF(btrim(city), '') IS NOT NULL AND canonical_city_id IS NULL;

    SELECT count(*) INTO unmapped_preferred_locations
    FROM jobhub.candidate_profiles p
    CROSS JOIN LATERAL unnest(p.preferred_locations) value
    WHERE jobhub.canonical_city_from_legacy(value) IS NULL;

    RAISE NOTICE 'canonical city backfill: % job locations, % profile cities, and % preferred locations remain unmapped',
        unmapped_jobs, unmapped_profile_cities, unmapped_preferred_locations;
END
$$;

CREATE INDEX jobs_canonical_city_public_idx ON jobhub.jobs (canonical_city_id)
WHERE canonical_city_id IS NOT NULL;

COMMENT ON COLUMN jobhub.jobs.canonical_city_id IS 'Canonical Kazakhstan city identifier; provider location_raw remains unchanged.';
COMMENT ON COLUMN jobhub.candidate_profiles.canonical_city_id IS 'Canonical professional profile city, separate from search preference.';
COMMENT ON COLUMN jobhub.candidate_profiles.preferred_city_ids IS 'Canonical preferred job-search cities.';

DROP FUNCTION jobhub.canonical_city_from_legacy(text);
