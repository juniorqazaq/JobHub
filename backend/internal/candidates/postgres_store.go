package candidates

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) GetProfile(ctx context.Context, userID string) (Profile, error) {
	const query = `
		SELECT u.id::text, u.full_name,
		       COALESCE(p.photo_url, ''), COALESCE(p.city, ''), p.birth_year, COALESCE(p.phone, ''), COALESCE(p.about, ''),
		       COALESCE(p.current_position, ''), COALESCE(p.desired_position, ''), p.years_experience::double precision,
		       COALESCE(p.experience_level, ''), COALESCE(p.certifications, '{}'), p.desired_salary::double precision,
		       COALESCE(p.currency, ''), COALESCE(p.salary_period, ''), COALESCE(p.preferred_locations, '{}'),
		       COALESCE(p.preferred_employment_types, '{}'), COALESCE(p.preferred_work_modes, '{}'),
		       COALESCE(p.preferred_categories, '{}'), COALESCE(p.preferred_roles, '{}'),
		       COALESCE(p.search_status, 'actively_looking'), COALESCE(p.github_url, ''), COALESCE(p.linkedin_url, ''),
		       COALESCE(p.portfolio_url, ''), COALESCE(p.website_url, ''), COALESCE(p.allow_employer_contact, false),
		       COALESCE(p.show_profile_to_employers, false), COALESCE(p.show_salary_expectations, false)
		FROM jobhub.users u
		LEFT JOIN jobhub.candidate_profiles p ON p.user_id = u.id
		WHERE u.id = $1::uuid AND u.role = 'job_seeker'`
	var p Profile
	err := s.pool.QueryRow(ctx, query, userID).Scan(
		&p.UserID, &p.FullName, &p.PhotoURL, &p.City, &p.BirthYear, &p.Phone, &p.About,
		&p.CurrentPosition, &p.DesiredPosition, &p.YearsExperience, &p.ExperienceLevel, &p.Certifications,
		&p.DesiredSalary, &p.Currency, &p.SalaryPeriod, &p.PreferredLocations, &p.PreferredEmploymentTypes,
		&p.PreferredWorkModes, &p.PreferredCategories, &p.PreferredRoles, &p.SearchStatus, &p.GitHub,
		&p.LinkedIn, &p.Portfolio, &p.Website, &p.AllowEmployerContact, &p.ShowProfileToEmployers,
		&p.ShowSalaryExpectations,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("get profile: %w", err)
	}
	p.Skills, err = s.profileSkills(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	p.WorkExperience, err = s.profileExperience(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	p.Education, err = s.profileEducation(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	p.Languages, err = s.profileLanguages(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	if resume, resumeErr := s.GetActiveResume(ctx, userID); resumeErr == nil {
		p.Resume = &resume
	} else if !errors.Is(resumeErr, ErrNotFound) {
		return Profile{}, resumeErr
	}
	p.Completion = profileCompletion(p)
	return p, nil
}

func (s *PostgresStore) UpdateProfile(ctx context.Context, userID string, p Profile) (Profile, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Profile{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, item := range p.WorkExperience {
		if item.ID != "" && !ownedRow(ctx, tx, "work_experience", item.ID, userID) {
			return Profile{}, ErrNotFound
		}
	}
	for _, item := range p.Education {
		if item.ID != "" && !ownedRow(ctx, tx, "education", item.ID, userID) {
			return Profile{}, ErrNotFound
		}
	}
	for _, item := range p.Languages {
		if item.ID != "" && !ownedRow(ctx, tx, "candidate_languages", item.ID, userID) {
			return Profile{}, ErrNotFound
		}
	}
	if tag, err := tx.Exec(ctx, `UPDATE jobhub.users SET full_name=$2, updated_at=now() WHERE id=$1::uuid AND role='job_seeker'`, userID, p.FullName); err != nil || tag.RowsAffected() != 1 {
		if err != nil {
			return Profile{}, fmt.Errorf("update profile identity: %w", err)
		}
		return Profile{}, ErrNotFound
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO jobhub.candidate_profiles (
			user_id, photo_url, city, birth_year, phone, about, current_position, desired_position,
			years_experience, experience_level, certifications, desired_salary, currency, salary_period,
			preferred_locations, preferred_employment_types, preferred_work_modes, preferred_categories,
			preferred_roles, search_status, github_url, linkedin_url, portfolio_url, website_url,
			allow_employer_contact, show_profile_to_employers, show_salary_expectations
		) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)
		ON CONFLICT (user_id) DO UPDATE SET
			photo_url=EXCLUDED.photo_url, city=EXCLUDED.city, birth_year=EXCLUDED.birth_year, phone=EXCLUDED.phone,
			about=EXCLUDED.about, current_position=EXCLUDED.current_position, desired_position=EXCLUDED.desired_position,
			years_experience=EXCLUDED.years_experience, experience_level=EXCLUDED.experience_level,
			certifications=EXCLUDED.certifications, desired_salary=EXCLUDED.desired_salary, currency=EXCLUDED.currency,
			salary_period=EXCLUDED.salary_period, preferred_locations=EXCLUDED.preferred_locations,
			preferred_employment_types=EXCLUDED.preferred_employment_types, preferred_work_modes=EXCLUDED.preferred_work_modes,
			preferred_categories=EXCLUDED.preferred_categories, preferred_roles=EXCLUDED.preferred_roles,
			search_status=EXCLUDED.search_status, github_url=EXCLUDED.github_url, linkedin_url=EXCLUDED.linkedin_url,
			portfolio_url=EXCLUDED.portfolio_url, website_url=EXCLUDED.website_url,
			allow_employer_contact=EXCLUDED.allow_employer_contact, show_profile_to_employers=EXCLUDED.show_profile_to_employers,
			show_salary_expectations=EXCLUDED.show_salary_expectations, updated_at=now()`,
		userID, nilIfBlank(p.PhotoURL), nilIfBlank(p.City), p.BirthYear, nilIfBlank(p.Phone), nilIfBlank(p.About),
		nilIfBlank(p.CurrentPosition), nilIfBlank(p.DesiredPosition), p.YearsExperience, nilIfBlank(p.ExperienceLevel),
		p.Certifications, p.DesiredSalary, nilIfBlank(p.Currency), nilIfBlank(p.SalaryPeriod), p.PreferredLocations,
		p.PreferredEmploymentTypes, p.PreferredWorkModes, p.PreferredCategories, p.PreferredRoles, p.SearchStatus,
		nilIfBlank(p.GitHub), nilIfBlank(p.LinkedIn), nilIfBlank(p.Portfolio), nilIfBlank(p.Website),
		p.AllowEmployerContact, p.ShowProfileToEmployers, p.ShowSalaryExpectations,
	)
	if err != nil {
		return Profile{}, fmt.Errorf("upsert profile: %w", err)
	}
	if err := replaceProfileRows(ctx, tx, userID, p); err != nil {
		return Profile{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Profile{}, err
	}
	return s.GetProfile(ctx, userID)
}

func replaceProfileRows(ctx context.Context, tx pgx.Tx, userID string, p Profile) error {
	if _, err := tx.Exec(ctx, `DELETE FROM jobhub.work_experience WHERE candidate_id=$1::uuid`, userID); err != nil {
		return err
	}
	for _, item := range p.WorkExperience {
		idExpr := "gen_random_uuid()"
		args := []any{userID, item.Company, item.Position, item.EmploymentType, item.StartDate, nilIfBlank(item.EndDate), item.IsCurrent, nilIfBlank(item.Description), nilIfBlank(item.Achievements), item.Skills}
		if item.ID != "" {
			idExpr = "$11::uuid"
			args = append(args, item.ID)
		}
		query := `INSERT INTO jobhub.work_experience (id,candidate_id,company,position,employment_type,start_date,end_date,is_current,description,achievements,skills) VALUES (` + idExpr + `,$1::uuid,$2,$3,$4,$5::date,$6::date,$7,$8,$9,$10)`
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insert experience: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM jobhub.education WHERE candidate_id=$1::uuid`, userID); err != nil {
		return err
	}
	for _, item := range p.Education {
		idExpr := "gen_random_uuid()"
		args := []any{userID, item.Institution, item.Degree, item.FieldOfStudy, item.StartYear, item.GraduationYear, nilIfBlank(item.Description)}
		if item.ID != "" {
			idExpr = "$8::uuid"
			args = append(args, item.ID)
		}
		query := `INSERT INTO jobhub.education (id,candidate_id,institution,degree,field_of_study,start_year,graduation_year,description) VALUES (` + idExpr + `,$1::uuid,$2,$3,$4,$5,$6,$7)`
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insert education: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM jobhub.candidate_languages WHERE candidate_id=$1::uuid`, userID); err != nil {
		return err
	}
	for _, item := range p.Languages {
		idExpr := "gen_random_uuid()"
		args := []any{userID, item.Language, normalize(item.Language), item.Proficiency}
		if item.ID != "" {
			idExpr = "$5::uuid"
			args = append(args, item.ID)
		}
		query := `INSERT INTO jobhub.candidate_languages (id,candidate_id,language,normalized_language,proficiency) VALUES (` + idExpr + `,$1::uuid,$2,$3,$4)`
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insert language: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM jobhub.candidate_skills WHERE candidate_id=$1::uuid`, userID); err != nil {
		return err
	}
	for _, skill := range p.Skills {
		if _, err := tx.Exec(ctx, `
			WITH selected AS (
				INSERT INTO jobhub.skills (name, normalized_name) VALUES ($2,$3)
				ON CONFLICT (normalized_name) DO UPDATE SET name=jobhub.skills.name RETURNING id
			)
			INSERT INTO jobhub.candidate_skills (candidate_id, skill_id)
			SELECT $1::uuid,id FROM selected ON CONFLICT DO NOTHING`, userID, skill, normalize(skill)); err != nil {
			return fmt.Errorf("insert skill: %w", err)
		}
	}
	return nil
}

func ownedRow(ctx context.Context, tx pgx.Tx, table, id, userID string) bool {
	query := `SELECT EXISTS (SELECT 1 FROM jobhub.` + table + ` WHERE id=$1::uuid AND candidate_id=$2::uuid)`
	var exists bool
	return tx.QueryRow(ctx, query, id, userID).Scan(&exists) == nil && exists
}

func (s *PostgresStore) profileSkills(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT sk.name FROM jobhub.candidate_skills cs JOIN jobhub.skills sk ON sk.id=cs.skill_id WHERE cs.candidate_id=$1::uuid ORDER BY lower(sk.name)`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		items = append(items, value)
	}
	return items, rows.Err()
}

func (s *PostgresStore) profileExperience(ctx context.Context, userID string) ([]WorkExperience, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text,company,position,employment_type,start_date,end_date,is_current,COALESCE(description,''),COALESCE(achievements,''),skills FROM jobhub.work_experience WHERE candidate_id=$1::uuid ORDER BY start_date DESC,id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []WorkExperience{}
	for rows.Next() {
		var item WorkExperience
		var start time.Time
		var end *time.Time
		if err := rows.Scan(&item.ID, &item.Company, &item.Position, &item.EmploymentType, &start, &end, &item.IsCurrent, &item.Description, &item.Achievements, &item.Skills); err != nil {
			return nil, err
		}
		item.StartDate = start.Format("2006-01-02")
		if end != nil {
			item.EndDate = end.Format("2006-01-02")
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) profileEducation(ctx context.Context, userID string) ([]Education, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text,institution,degree,field_of_study,start_year,graduation_year,COALESCE(description,'') FROM jobhub.education WHERE candidate_id=$1::uuid ORDER BY start_year DESC,id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Education{}
	for rows.Next() {
		var item Education
		if err := rows.Scan(&item.ID, &item.Institution, &item.Degree, &item.FieldOfStudy, &item.StartYear, &item.GraduationYear, &item.Description); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) profileLanguages(ctx context.Context, userID string) ([]Language, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text,language,proficiency FROM jobhub.candidate_languages WHERE candidate_id=$1::uuid ORDER BY lower(language)`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Language{}
	for rows.Next() {
		var item Language
		if err := rows.Scan(&item.ID, &item.Language, &item.Proficiency); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) GetActiveResume(ctx context.Context, candidateID string) (Resume, error) {
	return scanResume(s.pool.QueryRow(ctx, `SELECT id::text,original_filename,content_type,size_bytes,uploaded_at,status,storage_key FROM jobhub.resumes WHERE candidate_id=$1::uuid AND status='ready'`, candidateID))
}

func (s *PostgresStore) ReplaceResume(ctx context.Context, candidateID string, resume Resume) (Resume, *Resume, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Resume{}, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var old *Resume
	if current, currentErr := scanResume(tx.QueryRow(ctx, `SELECT id::text,original_filename,content_type,size_bytes,uploaded_at,status,storage_key FROM jobhub.resumes WHERE candidate_id=$1::uuid AND status='ready' FOR UPDATE`, candidateID)); currentErr == nil {
		old = &current
	} else if !errors.Is(currentErr, ErrNotFound) {
		return Resume{}, nil, currentErr
	}
	if old != nil {
		if _, err = tx.Exec(ctx, `UPDATE jobhub.resumes SET status='retired',retired_at=now() WHERE id=$1::uuid`, old.ID); err != nil {
			return Resume{}, nil, err
		}
	}
	created, err := scanResume(tx.QueryRow(ctx, `INSERT INTO jobhub.resumes (candidate_id,storage_key,original_filename,content_type,size_bytes) SELECT $1::uuid,$2,$3,$4,$5 FROM jobhub.users WHERE id=$1::uuid AND role='job_seeker' AND status='active' RETURNING id::text,original_filename,content_type,size_bytes,uploaded_at,status,storage_key`, candidateID, resume.storageKey, resume.OriginalFilename, resume.ContentType, resume.SizeBytes))
	if err != nil {
		return Resume{}, nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Resume{}, nil, err
	}
	return created, old, nil
}

func (s *PostgresStore) DeleteResume(ctx context.Context, candidateID string) (*Resume, bool, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	resume, err := scanResume(tx.QueryRow(ctx, `SELECT id::text,original_filename,content_type,size_bytes,uploaded_at,status,storage_key FROM jobhub.resumes WHERE candidate_id=$1::uuid AND status='ready' FOR UPDATE`, candidateID))
	if err != nil {
		return nil, false, err
	}
	var referenced bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobhub.applications WHERE resume_id=$1::uuid)`, resume.ID).Scan(&referenced); err != nil {
		return nil, false, err
	}
	if _, err = tx.Exec(ctx, `UPDATE jobhub.resumes SET status='deleted',deleted_at=now() WHERE id=$1::uuid`, resume.ID); err != nil {
		return nil, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, false, err
	}
	return &resume, !referenced, nil
}

func scanResume(row pgx.Row) (Resume, error) {
	var r Resume
	err := row.Scan(&r.ID, &r.OriginalFilename, &r.ContentType, &r.SizeBytes, &r.UploadedAt, &r.Status, &r.storageKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return Resume{}, ErrNotFound
	}
	if err != nil {
		return Resume{}, err
	}
	return r, nil
}

func profileCompletion(p Profile) Completion {
	missing := []string{}
	if p.DesiredPosition == "" {
		missing = append(missing, "desired_position")
	}
	if p.City == "" && len(p.PreferredLocations) == 0 {
		missing = append(missing, "location")
	}
	if len(p.WorkExperience) == 0 {
		missing = append(missing, "work_experience")
	}
	if len(p.Skills) == 0 {
		missing = append(missing, "skills")
	}
	if len(p.Education) == 0 {
		missing = append(missing, "education")
	}
	if len(p.PreferredWorkModes) == 0 {
		missing = append(missing, "preferred_work_mode")
	}
	if p.DesiredSalary == nil {
		missing = append(missing, "desired_salary")
	}
	if p.Resume == nil {
		missing = append(missing, "resume")
	}
	return Completion{Percentage: (8 - len(missing)) * 100 / 8, Missing: missing}
}

func normalize(value string) string { return strings.ToLower(strings.Join(strings.Fields(value), " ")) }
func nilIfBlank(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func uniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
