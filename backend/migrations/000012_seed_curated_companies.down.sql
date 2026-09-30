DELETE FROM jobhub.companies
WHERE name IN ('Air Astana', 'Kaspi.kz', 'Halyk Bank')
  AND NOT EXISTS (SELECT 1 FROM jobhub.jobs WHERE company_id = jobhub.companies.id)
  AND NOT EXISTS (SELECT 1 FROM jobhub.company_follows WHERE company_id = jobhub.companies.id)
  AND NOT EXISTS (SELECT 1 FROM jobhub.company_memberships WHERE company_id = jobhub.companies.id);
