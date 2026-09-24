package candidates

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresStore) SaveJob(ctx context.Context, candidateID, jobID string) error {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO jobhub.saved_jobs (candidate_id,job_id)
		SELECT u.id,j.id FROM jobhub.users u JOIN jobhub.jobs j ON j.id=$2::uuid
		JOIN jobhub.job_sources src ON src.source=j.source
		WHERE u.id=$1::uuid AND u.role='job_seeker' AND u.status='active' AND src.enabled AND jobhub.job_is_public(j)
		ON CONFLICT (candidate_id,job_id) DO NOTHING`, candidateID, jobID)
	if err != nil {
		return fmt.Errorf("save job: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	var exists bool
	if err = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobhub.saved_jobs WHERE candidate_id=$1::uuid AND job_id=$2::uuid)`, candidateID, jobID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	return ErrJobUnavailable
}

func (s *PostgresStore) UnsaveJob(ctx context.Context, candidateID, jobID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM jobhub.saved_jobs WHERE candidate_id=$1::uuid AND job_id=$2::uuid`, candidateID, jobID)
	return err
}

func (s *PostgresStore) ListSavedJobs(ctx context.Context, candidateID string) ([]SavedJob, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT sj.job_id::text,sj.created_at,jobhub.job_is_public(j),
		CASE WHEN jobhub.job_is_public(j) THEN j.title ELSE '' END,
		CASE WHEN jobhub.job_is_public(j) THEN COALESCE(j.company_name_raw,'') ELSE '' END,
		CASE WHEN jobhub.job_is_public(j) THEN COALESCE(j.location_raw,'') ELSE '' END,
		CASE WHEN jobhub.job_is_public(j) THEN j.source ELSE '' END,
		CASE WHEN jobhub.job_is_public(j) THEN src.display_name ELSE '' END,
		CASE WHEN jobhub.job_is_public(j) THEN j.application_method ELSE '' END
		FROM jobhub.saved_jobs sj JOIN jobhub.jobs j ON j.id=sj.job_id JOIN jobhub.job_sources src ON src.source=j.source
		WHERE sj.candidate_id=$1::uuid ORDER BY sj.created_at DESC`, candidateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SavedJob{}
	for rows.Next() {
		var item SavedJob
		if err := rows.Scan(&item.JobID, &item.SavedAt, &item.Available, &item.Title, &item.CompanyName, &item.Location, &item.Source, &item.SourceName, &item.Application); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateApplication(ctx context.Context, candidateID, jobID, resumeID, message string) (Application, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Application{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var source, companyID string
	var public bool
	err = tx.QueryRow(ctx, `SELECT source,COALESCE(company_id::text,''),jobhub.job_is_public(j) FROM jobhub.jobs j WHERE id=$1::uuid FOR UPDATE`, jobID).Scan(&source, &companyID, &public)
	if errors.Is(err, pgx.ErrNoRows) || !public {
		return Application{}, ErrJobUnavailable
	}
	if err != nil {
		return Application{}, err
	}
	if source != "jobhub" {
		return Application{}, ErrImportedJob
	}
	if companyID == "" {
		return Application{}, ErrJobUnavailable
	}
	var ready bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobhub.resumes WHERE id=$1::uuid AND candidate_id=$2::uuid AND status='ready')`, resumeID, candidateID).Scan(&ready); err != nil {
		return Application{}, err
	}
	if !ready {
		return Application{}, ErrResumeUnavailable
	}
	var applicationID string
	err = tx.QueryRow(ctx, `
		INSERT INTO jobhub.applications (job_id,company_id,candidate_id,resume_id,message)
		SELECT $1::uuid,$2::uuid,u.id,$3::uuid,NULLIF(btrim($4),'') FROM jobhub.users u
		WHERE u.id=$5::uuid AND u.role='job_seeker' AND u.status='active'
		RETURNING id::text`, jobID, companyID, resumeID, message, candidateID).Scan(&applicationID)
	if uniqueViolation(err) {
		_ = tx.Rollback(ctx)
		existing, lookupErr := s.getCandidateApplicationByJob(ctx, candidateID, jobID)
		if lookupErr != nil {
			return Application{}, ErrConflict
		}
		return existing, ErrConflict
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Application{}, ErrJobUnavailable
	}
	if err != nil {
		return Application{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO jobhub.application_status_events (application_id,actor_user_id,previous_status,new_status) VALUES ($1::uuid,$2::uuid,NULL,'sent')`, applicationID, candidateID); err != nil {
		return Application{}, err
	}
	application, err := getApplication(ctx, tx, `a.id=$1::uuid`, applicationID)
	if err != nil {
		return Application{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Application{}, err
	}
	return application, nil
}

func (s *PostgresStore) ListCandidateApplications(ctx context.Context, candidateID string) ([]Application, error) {
	rows, err := s.pool.Query(ctx, applicationSelect+` WHERE a.candidate_id=$1::uuid ORDER BY a.created_at DESC`, candidateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApplications(rows)
}

func (s *PostgresStore) getCandidateApplicationByJob(ctx context.Context, candidateID, jobID string) (Application, error) {
	return getApplication(ctx, s.pool, `a.candidate_id=$1::uuid AND a.job_id=$2::uuid`, candidateID, jobID)
}

func (s *PostgresStore) WithdrawApplication(ctx context.Context, candidateID, applicationID string) (Application, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Application{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM jobhub.applications WHERE id=$1::uuid AND candidate_id=$2::uuid FOR UPDATE`, applicationID, candidateID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return Application{}, ErrNotFound
	}
	if err != nil {
		return Application{}, err
	}
	if status == "rejected" || status == "withdrawn" {
		return Application{}, ErrInvalidState
	}
	if _, err = tx.Exec(ctx, `UPDATE jobhub.applications SET status='withdrawn',updated_at=now() WHERE id=$1::uuid`, applicationID); err != nil {
		return Application{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO jobhub.application_status_events(application_id,actor_user_id,previous_status,new_status) VALUES($1::uuid,$2::uuid,$3,'withdrawn')`, applicationID, candidateID, status); err != nil {
		return Application{}, err
	}
	item, err := getApplication(ctx, tx, `a.id=$1::uuid`, applicationID)
	if err != nil {
		return Application{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Application{}, err
	}
	return item, nil
}

func (s *PostgresStore) ListEmployerApplications(ctx context.Context, employerID, jobID string) ([]Application, error) {
	var owns bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobhub.jobs j JOIN jobhub.company_memberships cm ON cm.company_id=j.company_id WHERE j.id=$1::uuid AND j.source='jobhub' AND cm.user_id=$2::uuid AND cm.role='owner')`, jobID, employerID).Scan(&owns); err != nil {
		return nil, err
	}
	if !owns {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, applicationSelect+` WHERE a.job_id=$1::uuid AND EXISTS(SELECT 1 FROM jobhub.company_memberships cm WHERE cm.company_id=a.company_id AND cm.user_id=$2::uuid AND cm.role='owner') ORDER BY a.created_at DESC`, jobID, employerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApplications(rows)
}

func (s *PostgresStore) UpdateEmployerApplication(ctx context.Context, employerID, applicationID, newStatus string) (Application, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Application{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var current string
	err = tx.QueryRow(ctx, `SELECT a.status FROM jobhub.applications a WHERE a.id=$1::uuid AND EXISTS(SELECT 1 FROM jobhub.company_memberships cm WHERE cm.company_id=a.company_id AND cm.user_id=$2::uuid AND cm.role='owner') FOR UPDATE`, applicationID, employerID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return Application{}, ErrNotFound
	}
	if err != nil {
		return Application{}, err
	}
	if !validEmployerTransition(current, newStatus) {
		return Application{}, ErrInvalidState
	}
	if current != newStatus {
		if _, err = tx.Exec(ctx, `UPDATE jobhub.applications SET status=$2,updated_at=now() WHERE id=$1::uuid`, applicationID, newStatus); err != nil {
			return Application{}, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO jobhub.application_status_events(application_id,actor_user_id,previous_status,new_status) VALUES($1::uuid,$2::uuid,$3,$4)`, applicationID, employerID, current, newStatus); err != nil {
			return Application{}, err
		}
	}
	item, err := getApplication(ctx, tx, `a.id=$1::uuid`, applicationID)
	if err != nil {
		return Application{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Application{}, err
	}
	return item, nil
}

func (s *PostgresStore) GetEmployerApplicationResume(ctx context.Context, employerID, applicationID string) (Resume, error) {
	return scanResume(s.pool.QueryRow(ctx, `
		SELECT r.id::text,r.original_filename,r.content_type,r.size_bytes,r.uploaded_at,r.status,r.storage_key
		FROM jobhub.applications a JOIN jobhub.resumes r ON r.id=a.resume_id
		WHERE a.id=$1::uuid AND EXISTS(SELECT 1 FROM jobhub.company_memberships cm WHERE cm.company_id=a.company_id AND cm.user_id=$2::uuid AND cm.role='owner')`, applicationID, employerID))
}

func validEmployerTransition(current, next string) bool {
	if current == next {
		return true
	}
	if next == "rejected" {
		return current != "rejected" && current != "withdrawn"
	}
	rank := map[string]int{"sent": 0, "viewed": 1, "in_review": 2, "contacted": 3, "interview": 4, "offer": 5}
	currentRank, ok := rank[current]
	if !ok {
		return false
	}
	nextRank, ok := rank[next]
	return ok && nextRank > currentRank
}

const applicationSelect = `
	SELECT a.id::text,a.job_id::text,a.company_id::text,a.candidate_id::text,u.full_name,
	       COALESCE(p.desired_position,p.current_position,''),
	       CASE WHEN jobhub.job_is_public(j) THEN j.title ELSE '' END,
	       CASE WHEN jobhub.job_is_public(j) THEN c.name ELSE '' END,
	       jobhub.job_is_public(j),a.status,COALESCE(a.message,''),
	       r.id::text,r.original_filename,r.content_type,r.size_bytes,r.uploaded_at,r.status,r.storage_key,
	       a.created_at,a.updated_at,COALESCE((SELECT max(e.created_at) FROM jobhub.application_status_events e WHERE e.application_id=a.id),a.updated_at)
	FROM jobhub.applications a JOIN jobhub.jobs j ON j.id=a.job_id JOIN jobhub.companies c ON c.id=a.company_id
	JOIN jobhub.users u ON u.id=a.candidate_id LEFT JOIN jobhub.candidate_profiles p ON p.user_id=a.candidate_id
	JOIN jobhub.resumes r ON r.id=a.resume_id`

type rowQuery interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func getApplication(ctx context.Context, q rowQuery, where string, args ...any) (Application, error) {
	row := q.QueryRow(ctx, applicationSelect+` WHERE `+where, args...)
	return scanApplication(row)
}

func scanApplication(row pgx.Row) (Application, error) {
	var a Application
	err := row.Scan(&a.ID, &a.JobID, &a.CompanyID, &a.CandidateID, &a.CandidateName, &a.CandidatePosition, &a.JobTitle, &a.CompanyName, &a.JobAvailable, &a.Status, &a.Message, &a.Resume.ID, &a.Resume.OriginalFilename, &a.Resume.ContentType, &a.Resume.SizeBytes, &a.Resume.UploadedAt, &a.Resume.Status, &a.Resume.storageKey, &a.CreatedAt, &a.UpdatedAt, &a.LatestStatusAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Application{}, ErrNotFound
	}
	return a, err
}

func scanApplications(rows pgx.Rows) ([]Application, error) {
	items := []Application{}
	for rows.Next() {
		item, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func validStatus(value string) bool {
	return strings.Contains("|sent|viewed|in_review|contacted|interview|offer|rejected|", "|"+value+"|")
}
