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
