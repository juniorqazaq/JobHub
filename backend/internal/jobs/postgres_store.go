package jobs

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"jobhub-ai/backend/internal/database/dbgen"
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
	location := nullableText(strings.TrimSpace(params.Location))
	total, err := s.queries.CountPublicJobs(ctx, dbgen.CountPublicJobsParams{Query: query, Location: location, RequireProductionPermissions: s.requireProductionPermissions})
	if err != nil {
		return SearchResult{}, fmt.Errorf("count jobs: %w", err)
	}
	rows, err := s.queries.ListPublicJobs(ctx, dbgen.ListPublicJobsParams{
		Query: query, Location: location, RequireProductionPermissions: s.requireProductionPermissions,
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

func mapRow(row dbgen.JobhubJob) Job {
	job := Job{
		ID: formatUUID(row.ID), Source: row.Source, SourceName: sourceName(row.Source),
		ExternalID: textValue(row.ExternalID), SourceURL: textValue(row.SourceUrl),
		UpstreamSourceName: textValue(row.UpstreamSourceName), CompanyName: textValue(row.CompanyNameRaw),
		Title: row.Title, Location: textValue(row.LocationRaw), Description: textValue(row.Description),
		DescriptionKind: row.DescriptionKind, EmploymentType: textValue(row.EmploymentTypeRaw),
		SalaryRaw: textValue(row.SalaryRaw), ApplicationMethod: row.ApplicationMethod,
		ApplyURL: textValue(row.ApplyUrl), FirstSeenAt: row.FirstSeenAt.Time,
		LastSeenAt: row.LastSeenAt.Time, LastSyncedAt: row.LastSyncedAt.Time,
	}
	job.ExternalPublishedAt = timePtr(row.ExternalPublishedAt)
	job.ExternalUpdatedAt = timePtr(row.ExternalUpdatedAt)
	job.ExternalExpiresAt = timePtr(row.ExternalExpiresAt)
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

func formatUUID(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	b := value.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
