package jobs

import (
	"context"
	"errors"
	"time"
)

type SourceHealth struct {
	Source          string     `json:"source"`
	LastAttempt     *time.Time `json:"last_attempt"`
	LastSuccess     *time.Time `json:"last_success"`
	RequestCount    int        `json:"request_count"`
	JobsSeen        int        `json:"jobs_seen"`
	FailureCategory *string    `json:"failure_category"`
	Status          string     `json:"status"`
}

// SourceHealth is a read-only projection of real run records, not dry-run output.
// No cached registry columns or new schema are required for the bounded POC.
func (s *PostgresStore) SourceHealth(ctx context.Context, source string) (SourceHealth, error) {
	h := SourceHealth{Source: source}
	err := s.pool.QueryRow(ctx, `
 SELECT r.started_at,
   (SELECT max(finished_at) FROM jobhub.ingestion_runs WHERE source=$1 AND status='succeeded'),
   COALESCE(r.request_count,0),COALESCE(r.fetched_count,0),r.error_code,
   CASE WHEN NOT s.enabled THEN 'disabled'
        WHEN s.production_permissions_confirmed THEN 'permission_review_required'
        WHEN r.status='failed' THEN 'failing'
        WHEN r.status='succeeded' THEN 'healthy_development_only'
        WHEN r.status='running' THEN 'running'
        ELSE 'never_synced' END
 FROM jobhub.job_sources s
 LEFT JOIN LATERAL (SELECT * FROM jobhub.ingestion_runs WHERE source=s.source ORDER BY started_at DESC,id DESC LIMIT 1) r ON true
 WHERE s.source=$1`, source).Scan(&h.LastAttempt, &h.LastSuccess, &h.RequestCount, &h.JobsSeen, &h.FailureCategory, &h.Status)
	if err != nil {
		return h, errors.New("source health read failed")
	}
	return h, nil
}

// RegisterDevelopmentSource only inserts an unapproved local source. Conflicts
// are checked, not overwritten; it cannot enable an existing disabled source.
// The caller must first enforce the isolated-development database boundary.
func (s *PostgresStore) RegisterDevelopmentSource(ctx context.Context, source, provider, displayName string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO jobhub.job_sources(source,provider,display_name,enabled,production_permissions_confirmed) VALUES($1,$2,$3,true,false) ON CONFLICT(source) DO NOTHING`, source, provider, displayName)
	if err != nil {
		return errors.New("development source registration failed")
	}
	var allowed bool
	err = s.pool.QueryRow(ctx, `SELECT provider=$2 AND enabled AND NOT production_permissions_confirmed FROM jobhub.job_sources WHERE source=$1`, source, provider).Scan(&allowed)
	if err != nil || !allowed {
		return errors.New("source is disabled, incompatible, or approved for production; POC refused")
	}
	return nil
}
