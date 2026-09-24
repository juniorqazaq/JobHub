package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jobhub-ai/backend/internal/jobs"
)

type jobReaderStub struct{ item jobs.Job }

func (s jobReaderStub) Search(context.Context, jobs.SearchParams) (jobs.SearchResult, error) {
	return jobs.SearchResult{Items: []jobs.Job{s.item}, Total: 1}, nil
}

type capturingJobReader struct {
	params jobs.SearchParams
	calls  int
}

func (s *capturingJobReader) Search(_ context.Context, params jobs.SearchParams) (jobs.SearchResult, error) {
	s.params = params
	s.calls++
	return jobs.SearchResult{}, nil
}

func (s *capturingJobReader) Get(context.Context, string) (jobs.Job, error) {
	return jobs.Job{}, jobs.ErrNotFound
}
func (s jobReaderStub) Get(_ context.Context, id string) (jobs.Job, error) {
	if id != s.item.ID {
		return jobs.Job{}, jobs.ErrNotFound
	}
	return s.item, nil
}

func TestImportedJobEndpoints(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	reader := jobReaderStub{item: jobs.Job{
		ID: "11111111-1111-4111-8111-111111111111", Source: "jooble:kz", SourceName: "Jooble",
		SourceURL: "https://kz.jooble.org/jdp/123", CompanyName: "Example KZ", Title: "Go Developer",
		Location: "Алматы", Description: "Build services", DescriptionKind: "snippet", ApplicationMethod: "external",
		ApplyURL: "https://kz.jooble.org/jdp/123", FirstSeenAt: now, LastSeenAt: now, LastSyncedAt: now,
	}}
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", reader)
	for _, tc := range []struct{ path, expected string }{
		{"/api/v1/jobs", `"total":1`},
		{"/api/v1/jobs/11111111-1111-4111-8111-111111111111", `"method":"external"`},
	} {
		res := httptest.NewRecorder()
		router.ServeHTTP(res, httptest.NewRequest("GET", tc.path, nil))
		if res.Code != 200 || !strings.Contains(res.Body.String(), tc.expected) || !strings.Contains(res.Body.String(), `"id":"jooble:kz"`) {
			t.Fatalf("unexpected response for %s: %d %s", tc.path, res.Code, res.Body.String())
		}
	}
}

func TestListJobsRejectsUnsupportedSort(t *testing.T) {
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", jobReaderStub{})
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest("GET", "/api/v1/jobs?sort=salary", nil))
	if res.Code != 400 || !strings.Contains(res.Body.String(), `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("unexpected response: %d %s", res.Code, res.Body.String())
	}
}

func TestListJobsRejectsInvalidFiltersBeforeQueryingStore(t *testing.T) {
	tests := []string{
		"city=unknown",
		"preferred_city=unknown",
		"work_mode=remote,teleport",
		"salary_min=-1&currency=KZT",
		"salary_min=100&currency=GBP",
		"currency=KZT",
		"experience=principal",
		"employment=freelance",
		"date_posted=forever",
	}
	for _, query := range tests {
		t.Run(query, func(t *testing.T) {
			reader := &capturingJobReader{}
			router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", reader)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, httptest.NewRequest("GET", "/api/v1/jobs?"+query, nil))
			if res.Code != 400 || reader.calls != 0 {
				t.Fatalf("expected validation rejection before store call, got %d with %d calls: %s", res.Code, reader.calls, res.Body.String())
			}
		})
	}
}

func TestListJobsPassesCanonicalCombinedFiltersAndOpaqueQuery(t *testing.T) {
	reader := &capturingJobReader{}
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", reader)
	res := httptest.NewRecorder()
	path := "/api/v1/jobs?q=%27%3BDELETE+FROM+jobs%3B--&city=almaty&preferred_city=astana&work_mode=remote%2Chybrid&salary_min=500000&currency=kzt&experience=middle&employment=full_time&date_posted=7d&sort=oldest&page=2&page_size=25"
	router.ServeHTTP(res, httptest.NewRequest("GET", path, nil))
	if res.Code != 200 || reader.calls != 1 {
		t.Fatalf("unexpected response: %d %s", res.Code, res.Body.String())
	}
	p := reader.params
	if p.Query != "';DELETE FROM jobs;--" || p.City != "almaty" || p.PreferredCity != "astana" || len(p.WorkModes) != 2 || p.SalaryMin == nil || *p.SalaryMin != 500000 || p.Currency != "KZT" || p.ExperienceLevel != "middle" || p.EmploymentType != "full_time" || p.PostedAfter == nil || p.Sort != "oldest" || p.Page != 2 || p.PageSize != 25 {
		t.Fatalf("filters were not preserved: %#v", p)
	}
}

func TestPublicNativeJobHidesPrivateSalary(t *testing.T) {
	minimum, maximum := 100000.0, 300000.0
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	reader := jobReaderStub{item: jobs.Job{
		ID: "11111111-1111-4111-8111-111111111111", Source: "jobhub", SourceName: "JobHub",
		CompanyName: "Example", Title: "Developer", Location: "Almaty", Description: "Build services",
		DescriptionKind: "full", ApplicationMethod: "internal", FirstSeenAt: now, LastSeenAt: now, LastSyncedAt: now,
		SalaryMin: &minimum, SalaryMax: &maximum, SalaryCurrency: "USD", SalaryPeriod: "month", SalaryVisible: false,
	}}
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", reader)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest("GET", "/api/v1/jobs/11111111-1111-4111-8111-111111111111", nil))
	if res.Code != 200 || strings.Contains(res.Body.String(), "100000") || strings.Contains(res.Body.String(), `"salary_currency":"USD"`) {
		t.Fatalf("private salary leaked in public response: %d %s", res.Code, res.Body.String())
	}
}
