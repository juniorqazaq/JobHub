package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/providers/jooble"
)

type JoobleSearcher interface {
	Search(context.Context, jooble.Search) ([]jooble.Vacancy, error)
	RequestCount() int
}

type Service struct {
	store    jobs.Store
	provider JoobleSearcher
	logger   *slog.Logger
	freshFor time.Duration
	now      func() time.Time
}

func NewService(store jobs.Store, provider JoobleSearcher, logger *slog.Logger, freshFor time.Duration) *Service {
	return &Service{store: store, provider: provider, logger: logger, freshFor: freshFor, now: time.Now}
}

func (s *Service) RunJooble(ctx context.Context, searches []jooble.Search) (jobs.ImportStats, error) {
	stats := jobs.ImportStats{SearchCount: len(searches)}
	runID, err := s.store.BeginIngestionRun(ctx, jooble.Source, len(searches))
	if err != nil {
		return stats, err
	}
	stats.RunID = runID

	unique := make(map[string]jobs.ImportedJob)
	for _, search := range searches {
		vacancies, searchErr := s.provider.Search(ctx, search)
		stats.RequestCount = s.provider.RequestCount()
		if searchErr != nil {
			_ = s.store.FailIngestionRun(ctx, runID, stats, "PROVIDER_REQUEST_FAILED", "Jooble provider request failed")
			s.logger.Error("Jooble ingestion failed", "run_id", runID, "request_count", stats.RequestCount, "error", searchErr)
			return stats, searchErr
		}
		stats.FetchedCount += len(vacancies)
		for _, vacancy := range vacancies {
			item, normalizeErr := jooble.Normalize(vacancy)
			if normalizeErr != nil {
				stats.SkippedCount++
				continue
			}
			stats.NormalizedCount++
			unique[item.ExternalID] = item
		}
	}

	items := make([]jobs.ImportedJob, 0, len(unique))
	for _, item := range unique {
		items = append(items, item)
	}
	observedAt := s.now().UTC()
	stats, err = s.store.CompleteIngestionRun(ctx, runID, jooble.Source, items, observedAt, observedAt.Add(s.freshFor), stats)
	if err != nil {
		_ = s.store.FailIngestionRun(ctx, runID, stats, "DATABASE_WRITE_FAILED", "Atomic ingestion write failed")
		return stats, fmt.Errorf("store Jooble import: %w", err)
	}
	s.logger.Info("Jooble ingestion completed", "run_id", runID, "request_count", stats.RequestCount, "fetched", stats.FetchedCount, "normalized", stats.NormalizedCount, "inserted", stats.InsertedCount, "updated", stats.UpdatedCount, "skipped", stats.SkippedCount)
	return stats, nil
}
