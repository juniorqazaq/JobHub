INSERT INTO jobhub.companies (name, description, website_url, logo_url, industry, city, is_verified)
VALUES
    ('Air Astana', 'Қазақстанның әуе тасымалдаушысы.', 'https://airastana.com', '/company-logos/air-astana.svg', 'Авиация', 'Алматы', true),
    ('Kaspi.kz', 'Қазақстандағы технологиялық және қаржылық экожүйе.', 'https://kaspi.kz', NULL, 'Технологиялар және қаржы', 'Алматы', true),
    ('Halyk Bank', 'Қазақстандағы банк және қаржылық қызметтер тобы.', 'https://halykbank.kz', '/company-logos/halyk-bank.svg', 'Банктік қызметтер', 'Алматы', true)
ON CONFLICT (lower(regexp_replace(btrim(name), '\s+', ' ', 'g')))
DO UPDATE SET
    description = EXCLUDED.description,
    website_url = EXCLUDED.website_url,
    logo_url = EXCLUDED.logo_url,
    industry = EXCLUDED.industry,
    city = EXCLUDED.city,
    is_verified = true,
    updated_at = now();
