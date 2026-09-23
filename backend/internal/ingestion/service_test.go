package ingestion

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/providers/jooble"
)

type fakeProvider struct {
	vacancies []jooble.Vacancy
	err       error
	requests  int
}

func (p *fakeProvider) Search(context.Context, jooble.Search) ([]jooble.Vacancy, error) {
	p.requests++
	return p.vacancies, p.err
}
func (p *fakeProvider) RequestCount() int { return p.requests }

type storedJob struct {
	localID    string
	job        jobs.ImportedJob
	firstSeen  time.Time
	lastSeen   time.Time
	lastSynced time.Time
}

type memoryStore struct {
	items   map[string]storedJob
	runs    map[string]string
	nextRun int
}

func newMemoryStore() *memoryStore {
	return &memoryStore{items: map[string]storedJob{}, runs: map[string]string{}}
}
func (s *memoryStore) BeginIngestionRun(_ context.Context, _ string, _ int) (string, error) {
	s.nextRun++
	id := string(rune('0' + s.nextRun))
	s.runs[id] = "running"
	return id, nil
}
func (s *memoryStore) FailIngestionRun(_ context.Context, id string, _ jobs.ImportStats, _, _ string) error {
	s.runs[id] = "failed"
	return nil
}
func (s *memoryStore) CompleteIngestionRun(_ context.Context, id, source string, imported []jobs.ImportedJob, observedAt, _ time.Time, stats jobs.ImportStats) (jobs.ImportStats, error) {
	for _, item := range imported {
		key := source + "/" + item.ExternalID
		current, exists := s.items[key]
		if !exists {
			current = storedJob{localID: "local-1", firstSeen: observedAt}
			stats.InsertedCount++
		} else {
			stats.UpdatedCount++
		}
		current.job = item
		current.lastSeen = observedAt
		current.lastSynced = observedAt
		s.items[key] = current
	}
	s.runs[id] = "succeeded"
	return stats, nil
}
func (s *memoryStore) Search(context.Context, jobs.SearchParams) (jobs.SearchResult, error) {
	return jobs.SearchResult{}, nil
}
func (s *memoryStore) Get(context.Context, string) (jobs.Job, error) {
	return jobs.Job{}, jobs.ErrNotFound
}

func TestImportIsIdempotentUpdatesInPlaceAndFailureKeepsFreshness(t *testing.T) {
	store := newMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	base := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	search := []jooble.Search{{Keywords: "Go", Location: "Казахстан", Page: 1, ResultsPerPage: 10}}

	firstProvider := &fakeProvider{vacancies: []jooble.Vacancy{{ID: "42", Title: "Go Developer", Link: "https://kz.jooble.org/jdp/42"}}}
	first := NewService(store, firstProvider, logger, 72*time.Hour)
	first.now = func() time.Time { return base }
	stats, err := first.RunJooble(context.Background(), search)
	if err != nil || stats.InsertedCount != 1 {
		t.Fatalf("first import failed: %#v %v", stats, err)
	}

	secondProvider := &fakeProvider{vacancies: []jooble.Vacancy{{ID: "42", Title: "Senior Go Developer", Link: "https://kz.jooble.org/jdp/42"}}}
	second := NewService(store, secondProvider, logger, 72*time.Hour)
	second.now = func() time.Time { return base.Add(time.Hour) }
	stats, err = second.RunJooble(context.Background(), search)
	if err != nil || stats.UpdatedCount != 1 || stats.InsertedCount != 0 || len(store.items) != 1 {
		t.Fatalf("repeat import was not idempotent: %#v %v", stats, err)
	}
	beforeFailure := store.items["jooble:kz/42"]
	if beforeFailure.localID != "local-1" || beforeFailure.job.Title != "Senior Go Developer" || !beforeFailure.firstSeen.Equal(base) || !beforeFailure.lastSynced.Equal(base.Add(time.Hour)) {
		t.Fatalf("update did not preserve identity/freshness: %#v", beforeFailure)
	}

	failedProvider := &fakeProvider{err: errors.New("provider unavailable")}
	failed := NewService(store, failedProvider, logger, 72*time.Hour)
	failed.now = func() time.Time { return base.Add(2 * time.Hour) }
	if _, err := failed.RunJooble(context.Background(), search); err == nil {
		t.Fatal("expected failed import")
	}
	afterFailure := store.items["jooble:kz/42"]
	if afterFailure.job.Title != beforeFailure.job.Title || !afterFailure.lastSeen.Equal(beforeFailure.lastSeen) || !afterFailure.lastSynced.Equal(beforeFailure.lastSynced) {
		t.Fatal("failed import corrupted stored job freshness")
	}
}
