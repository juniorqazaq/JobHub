package jobs

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"jobhub-ai/backend/internal/database/dbgen"
	"jobhub-ai/backend/internal/locations"
)

type PostgresStore struct {
	pool                         *pgxpool.Pool
	queries                      *dbgen.Queries
	requireProductionPermissions bool
}

func NewPostgresStore(pool *pgxpool.Pool, requireProductionPermissions ...bool) *PostgresStore {
	requireConfirmed := len(requireProductionPermissions) > 0 && requireProductionPermissions[0]
	return &PostgresStore{pool: pool, queries: dbgen.New(pool), requireProductionPermissions: requireConfirmed}
}

func (s *PostgresStore) BeginIngestionRun(ctx context.Context, source string, searchCount int) (string, error) {
	id, err := s.queries.CreateIngestionRun(ctx, dbgen.CreateIngestionRunParams{Source: source, SearchCount: int32(searchCount)})
	if err != nil {
		return "", fmt.Errorf("create ingestion run: %w", err)
	}
	return formatUUID(id), nil
}

func (s *PostgresStore) FailIngestionRun(ctx context.Context, runID string, stats ImportStats, code, message string) error {
	id, err := parseUUID(runID)
	if err != nil {
		return err
	}
	return s.queries.FailIngestionRun(ctx, dbgen.FailIngestionRunParams{
		RequestCount: int32(stats.RequestCount), FetchedCount: int32(stats.FetchedCount),
		NormalizedCount: int32(stats.NormalizedCount), ErrorCode: nullableText(code),
		ErrorMessage: nullableText(message), ID: id,
	})
}

func (s *PostgresStore) CompleteIngestionRun(ctx context.Context, runID, source string, imported []ImportedJob, observedAt, freshUntil time.Time, stats ImportStats) (ImportStats, error) {
	id, err := parseUUID(runID)
	if err != nil {
		return stats, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return stats, fmt.Errorf("begin ingestion transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := dbgen.New(tx)
	if err := q.AcquireSourceIngestionLock(ctx, source); err != nil {
		return stats, fmt.Errorf("lock source ingestion: %w", err)
	}
	for _, item := range imported {
		existed, err := q.ImportedJobExists(ctx, dbgen.ImportedJobExistsParams{Source: source, ExternalID: nullableText(item.ExternalID)})
		if err != nil {
			return stats, fmt.Errorf("check imported job: %w", err)
		}
		_, err = q.UpsertImportedJob(ctx, dbgen.UpsertImportedJobParams{
			Source: source, ExternalID: nullableText(item.ExternalID), SourceUrl: nullableText(item.SourceURL),
			UpstreamSourceName: nullableText(item.UpstreamSourceName), CompanyNameRaw: nullableText(item.CompanyNameRaw),
			Title: item.Title, LocationRaw: nullableText(item.LocationRaw), Description: nullableText(item.Description),
			CanonicalCityID: nullableText(item.CanonicalCityID),
			DescriptionKind: item.DescriptionKind, EmploymentTypeRaw: nullableText(item.EmploymentTypeRaw),
			SalaryRaw: nullableText(item.SalaryRaw), ObservedAt: timestamptz(observedAt), FreshUntil: timestamptz(freshUntil),
			ExternalUpdatedAt: nullableTime(item.ExternalUpdatedAt), ExternalUpdatedRaw: nullableText(item.ExternalUpdatedRaw),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			stats.SkippedCount++
			continue
		}
		if err != nil {
			return stats, fmt.Errorf("upsert imported job: %w", err)
		}
		if existed {
			stats.UpdatedCount++
		} else {
			stats.InsertedCount++
		}
	}
	if err := q.CompleteIngestionRun(ctx, dbgen.CompleteIngestionRunParams{
		RequestCount: int32(stats.RequestCount), FetchedCount: int32(stats.FetchedCount),
		NormalizedCount: int32(stats.NormalizedCount), InsertedCount: int32(stats.InsertedCount),
		UpdatedCount: int32(stats.UpdatedCount), SkippedCount: int32(stats.SkippedCount), ID: id,
	}); err != nil {
		return stats, fmt.Errorf("complete ingestion run: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return stats, fmt.Errorf("commit ingestion transaction: %w", err)
	}
	return stats, nil
}

func (s *PostgresStore) Search(ctx context.Context, params SearchParams) (SearchResult, error) {
	query := nullableText(strings.TrimSpace(params.Query))
	filters := dbgen.CountPublicJobsParams{RequireProductionPermissions: s.requireProductionPermissions, Query: query,
		City: nullableText(params.City), WorkModes: params.WorkModes, SalaryMin: nullableNumeric(params.SalaryMin),
		Currency: nullableText(params.Currency), ExperienceLevel: nullableText(params.ExperienceLevel),
		EmploymentType: nullableText(params.EmploymentType), PostedAfter: nullableTime(params.PostedAfter)}
	total, err := s.queries.CountPublicJobs(ctx, filters)
	if err != nil {
		return SearchResult{}, fmt.Errorf("count jobs: %w", err)
	}
	rows, err := s.queries.ListPublicJobs(ctx, dbgen.ListPublicJobsParams{
		Query: filters.Query, City: filters.City, WorkModes: filters.WorkModes, SalaryMin: filters.SalaryMin,
		Currency: filters.Currency, ExperienceLevel: filters.ExperienceLevel, EmploymentType: filters.EmploymentType,
		PostedAfter: filters.PostedAfter, PreferredCity: nullableText(params.PreferredCity), RequireProductionPermissions: s.requireProductionPermissions,
		Sort: params.Sort, PageOffset: int32((params.Page - 1) * params.PageSize), PageSize: int32(params.PageSize),
	})
	if err != nil {
		return SearchResult{}, fmt.Errorf("list jobs: %w", err)
	}
	items := make([]Job, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapRow(row))
	}
	return SearchResult{Items: items, Total: total}, nil
}

func (s *PostgresStore) Get(ctx context.Context, rawID string) (Job, error) {
	id, err := parseUUID(rawID)
	if err != nil {
		return Job{}, ErrNotFound
	}
	row, err := s.queries.GetPublicJob(ctx, dbgen.GetPublicJobParams{ID: id, RequireProductionPermissions: s.requireProductionPermissions})
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, fmt.Errorf("get job: %w", err)
	}
	return mapRow(row), nil
}

func (s *PostgresStore) ListForEmployer(ctx context.Context, rawUserID string) ([]Job, error) {
	userID, err := parseUUID(rawUserID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.queries.ListEmployerJobs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list employer jobs: %w", err)
	}
	items := make([]Job, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapRow(row))
	}
	return items, nil
}

func (s *PostgresStore) GetForEmployer(ctx context.Context, rawUserID, rawID string) (Job, error) {
	userID, id, err := parseActorAndJob(rawUserID, rawID)
	if err != nil {
		return Job{}, ErrNotFound
	}
	row, err := s.queries.GetEmployerJob(ctx, dbgen.GetEmployerJobParams{ID: id, UserID: userID})
	return mapOwnedRow(row, err, "get employer job")
}

func (s *PostgresStore) CreateForEmployer(ctx context.Context, rawUserID string, input NativeJobInput) (Job, error) {
	userID, err := parseUUID(rawUserID)
	if err != nil {
		return Job{}, ErrNotFound
	}
	row, err := s.queries.CreateNativeJob(ctx, dbgen.CreateNativeJobParams{
		UserID: userID, Title: input.Title, Category: nullableText(input.Category), Description: nullableText(input.Description),
		Responsibilities: nullableText(input.Responsibilities), Requirements: nullableText(input.Requirements), NiceToHave: nullableText(input.NiceToHave),
		Skills: input.Skills, Location: nullableText(locations.Name(input.CanonicalCityID, "en")), WorkMode: nullableText(input.WorkMode), EmploymentType: nullableText(input.EmploymentType),
		ExperienceLevel: nullableText(input.ExperienceLevel), SalaryMin: nullableNumeric(input.SalaryMin), SalaryMax: nullableNumeric(input.SalaryMax),
		SalaryCurrency: nullableText(input.SalaryCurrency), SalaryPeriod: nullableText(input.SalaryPeriod), SalaryVisible: input.SalaryVisible,
		Benefits: input.Benefits, ExpiresAt: nullableTime(input.ExpiresAt), CanonicalCityID: nullableText(input.CanonicalCityID),
	})
	return mapOwnedRow(row, err, "create employer job")
}

func (s *PostgresStore) UpdateForEmployer(ctx context.Context, rawUserID, rawID string, input NativeJobInput) (Job, error) {
	userID, id, err := parseActorAndJob(rawUserID, rawID)
	if err != nil {
		return Job{}, ErrNotFound
	}
	row, err := s.queries.UpdateNativeJob(ctx, dbgen.UpdateNativeJobParams{
		Title: input.Title, Category: nullableText(input.Category), Description: nullableText(input.Description), Responsibilities: nullableText(input.Responsibilities),
		Requirements: nullableText(input.Requirements), NiceToHave: nullableText(input.NiceToHave), Skills: input.Skills, Location: nullableText(locations.Name(input.CanonicalCityID, "en")),
		WorkMode: nullableText(input.WorkMode), EmploymentType: nullableText(input.EmploymentType), ExperienceLevel: nullableText(input.ExperienceLevel),
		SalaryMin: nullableNumeric(input.SalaryMin), SalaryMax: nullableNumeric(input.SalaryMax), SalaryCurrency: nullableText(input.SalaryCurrency),
		SalaryPeriod: nullableText(input.SalaryPeriod), SalaryVisible: input.SalaryVisible, Benefits: input.Benefits, ExpiresAt: nullableTime(input.ExpiresAt),
		ID: id, UserID: userID, CanonicalCityID: nullableText(input.CanonicalCityID),
	})
	return mapOwnedRow(row, err, "update employer job")
}

func (s *PostgresStore) TransitionForEmployer(ctx context.Context, rawUserID, rawID, status string) (Job, error) {
	userID, id, err := parseActorAndJob(rawUserID, rawID)
	if err != nil {
		return Job{}, ErrNotFound
	}
	row, err := s.queries.TransitionNativeJob(ctx, dbgen.TransitionNativeJobParams{PublicationStatus: nullableText(status), ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrInvalidTransition
	}
	if err != nil {
		return Job{}, fmt.Errorf("transition employer job: %w", err)
	}
	return mapRow(row), nil
}

func (s *PostgresStore) DeleteForEmployer(ctx context.Context, rawUserID, rawID string) error {
	userID, id, err := parseActorAndJob(rawUserID, rawID)
	if err != nil {
		return ErrNotFound
	}
	_, err = s.queries.SoftDeleteNativeJob(ctx, dbgen.SoftDeleteNativeJobParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("delete employer job: %w", err)
	}
	return nil
}

func mapOwnedRow(row dbgen.JobhubJob, err error, operation string) (Job, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, fmt.Errorf("%s: %w", operation, err)
	}
	return mapRow(row), nil
}

func mapRow(row dbgen.JobhubJob) Job {
	job := Job{
		ID: formatUUID(row.ID), Source: row.Source, SourceName: sourceName(row.Source),
		ExternalID: textValue(row.ExternalID), SourceURL: textValue(row.SourceUrl),
		UpstreamSourceName: textValue(row.UpstreamSourceName), CompanyName: textValue(row.CompanyNameRaw),
		CompanyID: formatUUID(row.CompanyID), Title: row.Title, Category: textValue(row.Category), Location: textValue(row.LocationRaw), CanonicalCityID: textValue(row.CanonicalCityID), Description: textValue(row.Description),
		Responsibilities: textValue(row.Responsibilities), Requirements: textValue(row.Requirements), NiceToHave: textValue(row.NiceToHave),
		Skills: row.Skills, WorkMode: textValue(row.WorkMode), ExperienceLevel: textValue(row.ExperienceLevel),
		DescriptionKind: row.DescriptionKind, EmploymentType: textValue(row.EmploymentTypeRaw),
		SalaryRaw: textValue(row.SalaryRaw), SalaryMin: numericValue(row.SalaryMin), SalaryMax: numericValue(row.SalaryMax),
		SalaryCurrency: textValue(row.SalaryCurrency), SalaryPeriod: textValue(row.SalaryPeriod), SalaryVisible: row.SalaryVisible,
		Benefits: row.Benefits, ApplicationMethod: row.ApplicationMethod, PublicationStatus: textValue(row.PublicationStatus), ModerationStatus: row.ModerationStatus,
		ApplyURL: textValue(row.ApplyUrl), FirstSeenAt: row.FirstSeenAt.Time,
		LastSeenAt: row.LastSeenAt.Time, LastSyncedAt: row.LastSyncedAt.Time, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
	if job.Source == "jobhub" {
		job.EmploymentType = textValue(row.EmploymentType)
	}
	job.ExternalPublishedAt = timePtr(row.ExternalPublishedAt)
	job.ExternalUpdatedAt = timePtr(row.ExternalUpdatedAt)
	job.ExternalExpiresAt = timePtr(row.ExternalExpiresAt)
	job.ExpiresAt = timePtr(row.ExpiresAt)
	job.PublishedAt = timePtr(row.PublishedAt)
	return job
}

func sourceName(source string) string {
	if source == "jooble:kz" {
		return "Jooble"
	}
	return "JobHub"
}

func nullableText(value string) pgtype.Text { return pgtype.Text{String: value, Valid: value != ""} }
func textValue(value pgtype.Text) string {
	if value.Valid {
		return value.String
	}
	return ""
}
func timestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}
func nullableTime(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return timestamptz(*value)
}
func nullableNumeric(value *float64) pgtype.Numeric {
	if value == nil {
		return pgtype.Numeric{}
	}
	var result pgtype.Numeric
	_ = result.Scan(strconv.FormatFloat(*value, 'f', -1, 64))
	return result
}
func numericValue(value pgtype.Numeric) *float64 {
	if !value.Valid {
		return nil
	}
	converted, err := value.Float64Value()
	if err != nil || !converted.Valid {
		return nil
	}
	result := converted.Float64
	return &result
}
func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func parseUUID(value string) (pgtype.UUID, error) {
	compact := strings.ReplaceAll(value, "-", "")
	decoded, err := hex.DecodeString(compact)
	if err != nil || len(decoded) != 16 {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID")
	}
	var bytes [16]byte
	copy(bytes[:], decoded)
	return pgtype.UUID{Bytes: bytes, Valid: true}, nil
}

func parseActorAndJob(rawUserID, rawID string) (pgtype.UUID, pgtype.UUID, error) {
	userID, err := parseUUID(rawUserID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	id, err := parseUUID(rawID)
	return userID, id, err
}

func formatUUID(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	b := value.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
