package ingestion

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"jobhub-ai/backend/internal/database"
	"jobhub-ai/backend/internal/handlers"
	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/providers/jooble"
)

type integrationHealth struct{}

func (integrationHealth) Check(context.Context) error { return nil }

func TestPostgresImportIsAtomicAndIdempotent(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, databaseURL, "cache_statement")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	store := jobs.NewPostgresStore(pool)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	externalID := fmt.Sprintf("jobhub-poc-test-%d", time.Now().UnixNano())
	link := "https://kz.jooble.org/jdp/" + externalID
	searches := []jooble.Search{{Keywords: externalID, Location: "Казахстан", Page: 1, ResultsPerPage: 1}}
	var runIDs []string
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM jobhub.jobs WHERE source = 'jooble:kz' AND external_id = $1", externalID)
		for _, id := range runIDs {
			_, _ = pool.Exec(ctx, "DELETE FROM jobhub.ingestion_runs WHERE id = $1::uuid", id)
		}
	})

	firstTime := time.Now().UTC().Truncate(time.Microsecond)
	firstProvider := &fakeProvider{vacancies: []jooble.Vacancy{{ID: jooble.ExternalID(externalID), Title: "POC Go Developer " + externalID, Link: link, Location: "Алматы"}}}
	firstService := NewService(store, firstProvider, logger, 72*time.Hour)
	firstService.now = func() time.Time { return firstTime }
	firstStats, err := firstService.RunJooble(ctx, searches)
	if err != nil {
		t.Fatal(err)
	}
	runIDs = append(runIDs, firstStats.RunID)
	if firstStats.InsertedCount != 1 {
		t.Fatalf("expected insert, got %#v", firstStats)
	}
	firstResult, err := store.Search(ctx, jobs.SearchParams{Query: externalID, Page: 1, PageSize: 10})
	if err != nil || len(firstResult.Items) != 1 {
		t.Fatalf("stored job unavailable: %#v %v", firstResult, err)
	}
	firstJob := firstResult.Items[0]
	productionStore := jobs.NewPostgresStore(pool, true)
	productionResult, err := productionStore.Search(ctx, jobs.SearchParams{Query: externalID, Page: 1, PageSize: 10})
	if err != nil || productionResult.Total != 0 {
		t.Fatalf("unapproved source was visible in production mode: %#v %v", productionResult, err)
	}
	if _, err := productionStore.Get(ctx, firstJob.ID); !errors.Is(err, jobs.ErrNotFound) {
		t.Fatalf("unapproved source detail was visible in production mode: %v", err)
	}

	secondTime := firstTime.Add(time.Minute)
	secondTitle := "POC Senior Go Developer " + externalID
	secondProvider := &fakeProvider{vacancies: []jooble.Vacancy{{ID: jooble.ExternalID(externalID), Title: secondTitle, Link: link, Location: "Алматы"}}}
	secondService := NewService(store, secondProvider, logger, 72*time.Hour)
	secondService.now = func() time.Time { return secondTime }
	secondStats, err := secondService.RunJooble(ctx, searches)
	if err != nil {
		t.Fatal(err)
	}
	runIDs = append(runIDs, secondStats.RunID)
	if secondStats.UpdatedCount != 1 || secondStats.InsertedCount != 0 {
		t.Fatalf("expected update, got %#v", secondStats)
	}
	secondResult, err := store.Search(ctx, jobs.SearchParams{Query: externalID, Page: 1, PageSize: 10})
	if err != nil || len(secondResult.Items) != 1 {
		t.Fatalf("updated job unavailable: %#v %v", secondResult, err)
	}
	secondJob := secondResult.Items[0]
	if secondJob.ID != firstJob.ID || secondJob.Title != secondTitle || !secondJob.FirstSeenAt.Equal(firstTime) || !secondJob.LastSyncedAt.Equal(secondTime) {
		t.Fatalf("row identity or timestamps changed incorrectly: before=%#v after=%#v", firstJob, secondJob)
	}
	router := handlers.NewRouter(integrationHealth{}, logger, "http://localhost:5173", store)
	for _, tc := range []struct{ path, expected string }{
		{"/api/v1/jobs?q=" + externalID, `"total":1`},
		{"/api/v1/jobs/" + secondJob.ID, `"method":"external"`},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", tc.path, nil))
		if response.Code != 200 || !strings.Contains(response.Body.String(), tc.expected) || !strings.Contains(response.Body.String(), `"id":"jooble:kz"`) {
			t.Fatalf("unexpected API response for %s: %d %s", tc.path, response.Code, response.Body.String())
		}
	}

	failedProvider := &fakeProvider{err: fmt.Errorf("provider unavailable")}
	failedService := NewService(store, failedProvider, logger, 72*time.Hour)
	failedStats, err := failedService.RunJooble(ctx, searches)
	if err == nil {
		t.Fatal("expected failed import")
	}
	runIDs = append(runIDs, failedStats.RunID)
	var failedStatus string
	var failedRequests int
	if err := pool.QueryRow(ctx, "SELECT status, request_count FROM jobhub.ingestion_runs WHERE id = $1::uuid", failedStats.RunID).Scan(&failedStatus, &failedRequests); err != nil {
		t.Fatal(err)
	}
	if failedStatus != "failed" || failedRequests != 1 {
		t.Fatalf("failed run metadata is incorrect: status=%s requests=%d", failedStatus, failedRequests)
	}
	afterFailure, err := store.Get(ctx, secondJob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterFailure.Title != secondJob.Title || !afterFailure.LastSeenAt.Equal(secondJob.LastSeenAt) || !afterFailure.LastSyncedAt.Equal(secondJob.LastSyncedAt) {
		t.Fatal("failed import changed persisted job freshness")
	}
}
