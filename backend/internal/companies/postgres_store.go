package companies

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool                         *pgxpool.Pool
	requireProductionPermissions bool
}

func NewPostgresStore(pool *pgxpool.Pool, requireProductionPermissions bool) *PostgresStore {
	return &PostgresStore{pool: pool, requireProductionPermissions: requireProductionPermissions}
}

const publicCompanyPredicate = `
	c.status = 'active'
	AND (
		c.is_verified
		OR
		EXISTS (SELECT 1 FROM jobhub.company_memberships cm WHERE cm.company_id = c.id)
		OR EXISTS (
			SELECT 1 FROM jobhub.jobs j
			JOIN jobhub.job_sources s ON s.source = j.source
			WHERE j.company_id = c.id
			  AND s.enabled
			  AND (NOT $1::boolean OR s.production_permissions_confirmed)
			  AND jobhub.job_is_public(j)
		)
	)`

const companySelectColumns = `c.id, c.name, COALESCE(c.description, ''), COALESCE(c.website_url, ''),
	COALESCE(c.logo_url, ''), COALESCE(c.industry, ''), COALESCE(c.city, ''), c.is_verified,
	(SELECT count(*) FROM jobhub.jobs j JOIN jobhub.job_sources s ON s.source = j.source
	 WHERE j.company_id = c.id AND s.enabled AND (NOT $1::boolean OR s.production_permissions_confirmed) AND jobhub.job_is_public(j)) AS open_jobs_count,
	(SELECT count(*) FROM jobhub.company_follows f WHERE f.company_id = c.id), c.created_at, c.updated_at`

func (s *PostgresStore) Search(ctx context.Context, query string, page, pageSize int) (SearchResult, error) {
	pattern := "%" + strings.TrimSpace(query) + "%"
	countSQL := `SELECT count(*) FROM jobhub.companies c WHERE ` + publicCompanyPredicate + `
		AND ($2::text = '%%' OR c.name ILIKE $2 OR COALESCE(c.industry, '') ILIKE $2 OR COALESCE(c.city, '') ILIKE $2)`
	var total int64
	if err := s.pool.QueryRow(ctx, countSQL, s.requireProductionPermissions, pattern).Scan(&total); err != nil {
		return SearchResult{}, fmt.Errorf("count companies: %w", err)
	}
	listSQL := `SELECT ` + companySelectColumns + `
		FROM jobhub.companies c WHERE ` + publicCompanyPredicate + `
		AND ($2::text = '%%' OR c.name ILIKE $2 OR COALESCE(c.industry, '') ILIKE $2 OR COALESCE(c.city, '') ILIKE $2)
		ORDER BY c.is_verified DESC, open_jobs_count DESC, c.name ASC
		LIMIT $3 OFFSET $4`
	rows, err := s.pool.Query(ctx, listSQL, s.requireProductionPermissions, pattern, pageSize, (page-1)*pageSize)
	if err != nil {
		return SearchResult{}, fmt.Errorf("list companies: %w", err)
	}
	defer rows.Close()
	items := make([]Company, 0)
	for rows.Next() {
		item, err := scanCompany(rows)
		if err != nil {
			return SearchResult{}, fmt.Errorf("scan company: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return SearchResult{}, fmt.Errorf("list companies rows: %w", err)
	}
	return SearchResult{Items: items, Total: total}, nil
}

func (s *PostgresStore) ListFollowed(ctx context.Context, rawCandidateID string) ([]Company, error) {
	candidateID, err := parseUUID(rawCandidateID)
	if err != nil {
		return nil, ErrNotFound
	}
	query := `SELECT ` + companySelectColumns + `
		FROM jobhub.company_follows f
		JOIN jobhub.companies c ON c.id = f.company_id
		WHERE f.candidate_id = $2 AND ` + publicCompanyPredicate + `
		ORDER BY f.created_at DESC, c.name ASC`
	rows, err := s.pool.Query(ctx, query, s.requireProductionPermissions, candidateID)
	if err != nil {
		return nil, fmt.Errorf("list followed companies: %w", err)
	}
	defer rows.Close()
	items := make([]Company, 0)
	for rows.Next() {
		item, err := scanCompany(rows)
		if err != nil {
			return nil, fmt.Errorf("scan followed company: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list followed companies rows: %w", err)
	}
	return items, nil
}

func (s *PostgresStore) GetPublic(ctx context.Context, rawID string) (Company, error) {
	id, err := parseUUID(rawID)
	if err != nil {
		return Company{}, ErrNotFound
	}
	companySQL := `SELECT ` + companySelectColumns + `
		FROM jobhub.companies c WHERE c.id = $2 AND ` + publicCompanyPredicate
	company, err := scanCompany(s.pool.QueryRow(ctx, companySQL, s.requireProductionPermissions, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Company{}, ErrNotFound
	}
	if err != nil {
		return Company{}, fmt.Errorf("get company: %w", err)
	}
	return company, nil
}

func (s *PostgresStore) ListPublicJobs(ctx context.Context, rawID string, page, pageSize int) (VacancySearchResult, error) {
	id, err := parseUUID(rawID)
	if err != nil {
		return VacancySearchResult{}, ErrNotFound
	}
	jobsSQL := `SELECT j.id, j.title, COALESCE(j.location_raw, ''), COALESCE(j.canonical_city_id, ''),
		COALESCE(j.work_mode, ''), COALESCE(j.employment_type, j.employment_type_raw, ''),
		j.salary_min, j.salary_max, COALESCE(j.salary_currency, ''), COALESCE(j.salary_period, ''),
		j.salary_visible, COALESCE(j.published_at, j.external_published_at, j.first_seen_at), j.source, s.display_name,
		count(*) OVER()
		FROM jobhub.jobs j JOIN jobhub.job_sources s ON s.source = j.source
		WHERE j.company_id = $2 AND s.enabled
		  AND (NOT $1::boolean OR s.production_permissions_confirmed)
		  AND jobhub.job_is_public(j)
		ORDER BY COALESCE(j.published_at, j.external_published_at, j.first_seen_at) DESC, j.id DESC
		LIMIT $3 OFFSET $4`
	rows, err := s.pool.Query(ctx, jobsSQL, s.requireProductionPermissions, id, pageSize, (page-1)*pageSize)
	if err != nil {
		return VacancySearchResult{}, fmt.Errorf("list company jobs: %w", err)
	}
	defer rows.Close()
	items := make([]Vacancy, 0)
	var total int64
	for rows.Next() {
		var item Vacancy
		var jobID pgtype.UUID
		var salaryMin, salaryMax pgtype.Numeric
		var published pgtype.Timestamptz
		if err := rows.Scan(&jobID, &item.Title, &item.Location, &item.CityID, &item.WorkMode, &item.EmploymentType,
			&salaryMin, &salaryMax, &item.SalaryCurrency, &item.SalaryPeriod, &item.SalaryVisible, &published, &item.Source, &item.SourceName, &total); err != nil {
			return VacancySearchResult{}, fmt.Errorf("scan company job: %w", err)
		}
		item.ID = formatUUID(jobID)
		item.SalaryMin = numericValue(salaryMin)
		item.SalaryMax = numericValue(salaryMax)
		if published.Valid {
			value := published.Time
			item.PublishedAt = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return VacancySearchResult{}, fmt.Errorf("list company jobs rows: %w", err)
	}
	if len(items) == 0 {
		company, err := s.GetPublic(ctx, rawID)
		if err != nil {
			return VacancySearchResult{}, err
		}
		total = company.OpenJobsCount
	}
	return VacancySearchResult{Items: items, Total: total}, nil
}

func (s *PostgresStore) GetForEmployer(ctx context.Context, rawUserID string) (Company, error) {
	userID, err := parseUUID(rawUserID)
	if err != nil {
		return Company{}, ErrNotFound
	}
	row := s.pool.QueryRow(ctx, `SELECT c.id, c.name, COALESCE(c.description, ''), COALESCE(c.website_url, ''),
		COALESCE(c.logo_url, ''), COALESCE(c.industry, ''), COALESCE(c.city, ''), c.is_verified,
		(SELECT count(*) FROM jobhub.jobs j WHERE j.company_id = c.id AND jobhub.job_is_public(j)),
		(SELECT count(*) FROM jobhub.company_follows f WHERE f.company_id = c.id), c.created_at, c.updated_at
		FROM jobhub.companies c JOIN jobhub.company_memberships cm ON cm.company_id = c.id
		WHERE cm.user_id = $1 AND cm.role = 'owner'`, userID)
	company, err := scanCompany(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Company{}, ErrNotFound
	}
	if err != nil {
		return Company{}, fmt.Errorf("get employer company: %w", err)
	}
	return company, nil
}

func (s *PostgresStore) UpdateForEmployer(ctx context.Context, rawUserID string, input UpdateInput) (Company, error) {
	userID, err := parseUUID(rawUserID)
	if err != nil {
		return Company{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Company{}, fmt.Errorf("begin company update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var companyID pgtype.UUID
	err = tx.QueryRow(ctx, `UPDATE jobhub.companies c SET name=$2, description=NULLIF($3,''), website_url=NULLIF($4,''),
		logo_url=NULLIF($5,''), industry=NULLIF($6,''), city=NULLIF($7,''), updated_at=now()
		WHERE EXISTS (SELECT 1 FROM jobhub.company_memberships cm WHERE cm.company_id=c.id AND cm.user_id=$1 AND cm.role='owner')
		RETURNING c.id`, userID, input.Name, input.Description, input.WebsiteURL, input.LogoURL, input.Industry, input.City).Scan(&companyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Company{}, ErrNotFound
	}
	if err != nil {
		return Company{}, fmt.Errorf("update company: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE jobhub.jobs SET company_name_raw=$2, updated_at=now() WHERE company_id=$1 AND source='jobhub'`, companyID, input.Name); err != nil {
		return Company{}, fmt.Errorf("update company job names: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Company{}, fmt.Errorf("commit company update: %w", err)
	}
	return s.GetForEmployer(ctx, rawUserID)
}

func (s *PostgresStore) IsFollowing(ctx context.Context, rawCandidateID, rawCompanyID string) (bool, error) {
	candidateID, companyID, err := parsePair(rawCandidateID, rawCompanyID)
	if err != nil {
		return false, ErrNotFound
	}
	var exists bool
	err = s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM jobhub.company_follows WHERE candidate_id=$1 AND company_id=$2)`, candidateID, companyID).Scan(&exists)
	return exists, err
}

func (s *PostgresStore) Follow(ctx context.Context, rawCandidateID, rawCompanyID string) error {
	candidateID, companyID, err := parsePair(rawCandidateID, rawCompanyID)
	if err != nil {
		return ErrNotFound
	}
	command, err := s.pool.Exec(ctx, `INSERT INTO jobhub.company_follows (candidate_id, company_id)
		SELECT $1, c.id FROM jobhub.companies c WHERE c.id=$2 AND c.status='active'
		ON CONFLICT DO NOTHING`, candidateID, companyID)
	if err != nil {
		return fmt.Errorf("follow company: %w", err)
	}
	if command.RowsAffected() == 0 {
		var exists bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM jobhub.companies WHERE id=$1 AND status='active')`, companyID).Scan(&exists); err != nil {
			return fmt.Errorf("check company: %w", err)
		}
		if !exists {
			return ErrNotFound
		}
	}
	return nil
}

func (s *PostgresStore) Unfollow(ctx context.Context, rawCandidateID, rawCompanyID string) error {
	candidateID, companyID, err := parsePair(rawCandidateID, rawCompanyID)
	if err != nil {
		return ErrNotFound
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM jobhub.company_follows WHERE candidate_id=$1 AND company_id=$2`, candidateID, companyID)
	return err
}

type scanner interface{ Scan(...any) error }

func scanCompany(row scanner) (Company, error) {
	var item Company
	var id pgtype.UUID
	var createdAt, updatedAt pgtype.Timestamptz
	err := row.Scan(&id, &item.Name, &item.Description, &item.WebsiteURL, &item.LogoURL, &item.Industry, &item.City,
		&item.Verified, &item.OpenJobsCount, &item.FollowerCount, &createdAt, &updatedAt)
	item.ID = formatUUID(id)
	if createdAt.Valid {
		item.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		item.UpdatedAt = updatedAt.Time
	}
	return item, err
}

func parsePair(first, second string) (pgtype.UUID, pgtype.UUID, error) {
	a, err := parseUUID(first)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	b, err := parseUUID(second)
	return a, b, err
}

func parseUUID(value string) (pgtype.UUID, error) {
	decoded, err := hex.DecodeString(strings.ReplaceAll(value, "-", ""))
	if err != nil || len(decoded) != 16 {
		return pgtype.UUID{}, errors.New("invalid UUID")
	}
	var bytes [16]byte
	copy(bytes[:], decoded)
	return pgtype.UUID{Bytes: bytes, Valid: true}, nil
}

func formatUUID(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	b := value.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func numericValue(value pgtype.Numeric) *float64 {
	if !value.Valid {
		return nil
	}
	converted, err := value.Float64Value()
	if err != nil || !converted.Valid {
		return nil
	}
	parsed, err := strconv.ParseFloat(strconv.FormatFloat(converted.Float64, 'f', -1, 64), 64)
	if err != nil {
		return nil
	}
	return &parsed
}
