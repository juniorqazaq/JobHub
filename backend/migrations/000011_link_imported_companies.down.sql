UPDATE jobhub.jobs
SET company_id = NULL
WHERE source <> 'jobhub';

DROP INDEX IF EXISTS jobhub.companies_normalized_name_key;
