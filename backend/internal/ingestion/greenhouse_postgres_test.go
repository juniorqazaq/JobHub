package ingestion

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"jobhub-ai/backend/internal/candidates"
	"jobhub-ai/backend/internal/config"
	"jobhub-ai/backend/internal/database"
	"jobhub-ai/backend/internal/handlers"
	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/providers/greenhouse"
)

type redirectToFixture struct{ target *url.URL }

func (r redirectToFixture) RoundTrip(req *http.Request) (*http.Response, error) {
	copied := req.Clone(req.Context())
	u := *req.URL
	u.Scheme = r.target.Scheme
	u.Host = r.target.Host
	copied.URL = &u
	return http.DefaultTransport.RoundTrip(copied)
}

func TestGreenhousePostgresPipelineAndReadOnlyDryRun(t *testing.T) {
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	if err := config.ValidateATSDevelopmentDatabase("development", raw); err != nil {
		t.Skip("Greenhouse integration requires isolated local jobhub_ats_poc_* database")
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, raw, "cache_statement")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	store := jobs.NewPostgresStore(pool)
	board := fmt.Sprintf("fixture-%d", time.Now().UnixNano())
	source := "greenhouse:" + board
	if err := store.RegisterDevelopmentSource(ctx, source, "greenhouse", "Fixture only"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM jobhub.jobs WHERE source=$1", source)
		_, _ = pool.Exec(ctx, "DELETE FROM jobhub.ingestion_runs WHERE source=$1", source)
		_, _ = pool.Exec(ctx, "DELETE FROM jobhub.job_sources WHERE source=$1", source)
	})
	var stage atomic.Int32
	var requests atomic.Int32
	original := "https://careers.example/jobs/9007199254740993?gh_src=original"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/v1/boards/"+board+"/jobs" || r.URL.Query().Get("content") != "true" {
			t.Error("wrong endpoint")
		}
		w.Header().Set("Content-Type", "application/json")
		switch stage.Load() {
		case 2:
			w.WriteHeader(500)
			fmt.Fprint(w, "SECRET upstream error")
		case 3:
			fmt.Fprint(w, `{"jobs":[],"meta":{"total":0}}`)
		default:
			title := "Engineer"
			if stage.Load() == 1 {
				title = "Senior Engineer"
			}
			fmt.Fprintf(w, `{"jobs":[{"id":9007199254740993,"internal_job_id":1,"title":%q,"absolute_url":%q,"content":"<p>Build software</p>"}],"meta":{"total":1}}`, title, original)
		}
	}))
	defer server.Close()
	target, _ := url.Parse(server.URL)
	budget, _ := greenhouse.NewBudget(12, 200)
	adapter, err := greenhouse.NewClient(board, budget, &http.Client{Transport: redirectToFixture{target}})
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	service := NewService(store, nil, slog.New(slog.NewTextHandler(&logs, nil)), 0)
	firstTime := time.Now().UTC().Truncate(time.Microsecond)
	service.now = func() time.Time { return firstTime }
	first, err := service.Run(ctx, adapter, RunOptions{NoAutomaticExpiry: true})
	if err != nil || first.Stats.InsertedCount != 1 || !first.CompleteSnapshot {
		t.Fatalf("first: %+v %v", first, err)
	}
	var id string
	var firstSeen, timeBefore time.Time
	if err := pool.QueryRow(ctx, "SELECT id::text,first_seen_at,last_synced_at FROM jobhub.jobs WHERE source=$1", source).Scan(&id, &firstSeen, &timeBefore); err != nil {
		t.Fatal(err)
	}
	stage.Store(1)
	service.now = func() time.Time { return firstTime.Add(time.Minute) }
	second, err := service.Run(ctx, adapter, RunOptions{NoAutomaticExpiry: true})
	if err != nil || second.Stats.InsertedCount != 0 || second.Stats.UpdatedCount != 1 {
		t.Fatalf("second: %+v %v", second, err)
	}
	job, err := store.Get(ctx, id)
	if err != nil || job.Title != "Senior Engineer" || job.ID != id || !job.FirstSeenAt.Equal(firstSeen) || !job.LastSyncedAt.Equal(firstTime.Add(time.Minute)) || job.ApplyURL != original || job.ApplicationMethod != "external" || job.ExternalID != "9007199254740993" {
		t.Fatalf("identity/URL: %+v %v", job, err)
	}
	var count int
	var nullable bool
	if err := pool.QueryRow(ctx, `SELECT count(*),bool_and(salary_min IS NULL AND salary_max IS NULL AND employment_type_raw IS NULL AND employment_type IS NULL AND work_mode IS NULL AND company_id IS NULL AND external_published_at IS NULL) FROM jobhub.jobs WHERE source=$1`, source).Scan(&count, &nullable); err != nil || count != 1 || !nullable {
		t.Fatalf("dedupe/null values: %d %v %v", count, nullable, err)
	}
	h, err := store.SourceHealth(ctx, source)
	if err != nil || h.LastAttempt == nil || h.LastSuccess == nil || h.RequestCount != 1 || h.JobsSeen != 1 || h.FailureCategory != nil {
		t.Fatalf("health: %+v %v", h, err)
	}
	lastSuccess := *h.LastSuccess
	// Real PostgreSQL in read-only mode, and no change to jobs or run records.
	ro, err := database.Connect(ctx, raw+"&default_transaction_read_only=on", "cache_statement")
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	if _, err := ro.Exec(ctx, "UPDATE jobhub.job_sources SET display_name=display_name WHERE source=$1", source); err == nil {
		t.Fatal("read-only guard not active")
	}
	readonlyService := NewService(jobs.NewPostgresStore(ro), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), 0)
	if _, err := readonlyService.Run(ctx, adapter, RunOptions{DryRun: true, Sample: 2, NoAutomaticExpiry: true}); err != nil {
		t.Fatal(err)
	}
	var runs int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM jobhub.ingestion_runs WHERE source=$1", source).Scan(&runs); err != nil || runs != 2 {
		t.Fatalf("dry-run wrote a run: %d %v", runs, err)
	}
	afterDry, err := store.Get(ctx, id)
	if err != nil || !afterDry.LastSyncedAt.Equal(job.LastSyncedAt) {
		t.Fatal("dry-run changed freshness")
	}
	// Failed HTTP collection leaves every persisted vacancy timestamp unchanged.
	stage.Store(2)
	failed, err := service.Run(ctx, adapter, RunOptions{NoAutomaticExpiry: true})
	if err == nil || failed.FailureCategory != "HTTP_FAILED" {
		t.Fatalf("failure: %+v %v", failed, err)
	}
	unchanged, err := store.Get(ctx, id)
	if err != nil || unchanged.Title != job.Title || !unchanged.LastSeenAt.Equal(job.LastSeenAt) || !unchanged.LastSyncedAt.Equal(job.LastSyncedAt) {
		t.Fatal("failed fetch changed vacancy")
	}
	h, err = store.SourceHealth(ctx, source)
	if err != nil || h.LastSuccess == nil || !h.LastSuccess.Equal(lastSuccess) || h.FailureCategory == nil || *h.FailureCategory != "HTTP_FAILED" || h.Status != "failing" || h.RequestCount != 1 {
		t.Fatalf("failed health: %+v %v", h, err)
	}
	// A successful empty snapshot is healthy and does not remove/refresh old jobs.
	stage.Store(3)
	empty, err := service.Run(ctx, adapter, RunOptions{NoAutomaticExpiry: true})
	if err != nil || !empty.CompleteSnapshot || empty.Stats.FetchedCount != 0 {
		t.Fatalf("empty: %+v %v", empty, err)
	}
	unchanged, err = store.Get(ctx, id)
	if err != nil || !unchanged.LastSyncedAt.Equal(job.LastSyncedAt) {
		t.Fatal("empty snapshot expired/refreshed job")
	}
	h, err = store.SourceHealth(ctx, source)
	if err != nil || h.JobsSeen != 0 || h.FailureCategory != nil || h.Status == "failing" {
		t.Fatalf("empty health: %+v %v", h, err)
	}
	if _, err := candidates.NewPostgresStore(pool).CreateApplication(ctx, "", id, "", ""); !errors.Is(err, candidates.ErrImportedJob) {
		t.Fatalf("ATS job accepted internal application: %v", err)
	}
	var applications int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM jobhub.applications WHERE job_id=$1::uuid", id).Scan(&applications); err != nil || applications != 0 {
		t.Fatal("internal application created")
	}
	production := jobs.NewPostgresStore(pool, true)
	if _, err := production.Get(ctx, id); !errors.Is(err, jobs.ErrNotFound) {
		t.Fatal("unapproved ATS visible in production")
	}
	result, err := production.Search(ctx, jobs.SearchParams{Query: "Senior Engineer", Page: 1, PageSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range result.Items {
		if j.ID == id {
			t.Fatal("unapproved ATS in production list")
		}
	}
	router := handlers.NewRouter(integrationHealth{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", store)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/jobs/"+id, nil))
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"method":"external"`) || !strings.Contains(response.Body.String(), original) {
		t.Fatalf("external API: %s", response.Body.String())
	}
	if strings.Contains(logs.String(), "SECRET") {
		t.Fatal("secret in logs")
	}
	t.Logf("fixture HTTP requests=%d; first inserts=%d; second updates=%d; rows=%d; UUID preserved; dry-run read-only; failure/empty freshness unchanged", requests.Load(), first.Stats.InsertedCount, second.Stats.UpdatedCount, count)
}
