package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jobhub-ai/backend/internal/auth"
	"jobhub-ai/backend/internal/companies"
)

type companyStoreStub struct {
	query   string
	page    int
	updated companies.UpdateInput
}

func (s *companyStoreStub) Search(_ context.Context, query string, page, _ int) (companies.SearchResult, error) {
	s.query, s.page = query, page
	return companies.SearchResult{Items: []companies.Company{{ID: "11111111-1111-4111-8111-111111111111", Name: "Example", Verified: true, OpenJobsCount: 2}}, Total: 1}, nil
}
func (s *companyStoreStub) GetPublic(_ context.Context, id string) (companies.Company, error) {
	if id != "11111111-1111-4111-8111-111111111111" {
		return companies.Company{}, companies.ErrNotFound
	}
	return companies.Company{ID: id, Name: "Example", Verified: true, OpenJobsCount: 1}, nil
}
func (s *companyStoreStub) ListPublicJobs(_ context.Context, id string, page, pageSize int) (companies.VacancySearchResult, error) {
	if id != "11111111-1111-4111-8111-111111111111" {
		return companies.VacancySearchResult{}, companies.ErrNotFound
	}
	if page != 1 || pageSize != 20 {
		return companies.VacancySearchResult{}, errors.New("unexpected pagination")
	}
	return companies.VacancySearchResult{Items: []companies.Vacancy{{ID: "22222222-2222-4222-8222-222222222222", Title: "Developer"}}, Total: 1}, nil
}
func (s *companyStoreStub) ListFollowed(context.Context, string) ([]companies.Company, error) {
	return []companies.Company{{ID: "11111111-1111-4111-8111-111111111111", Name: "Example", Verified: true}}, nil
}
func (s *companyStoreStub) GetForEmployer(context.Context, string) (companies.Company, error) {
	return companies.Company{ID: "11111111-1111-4111-8111-111111111111", Name: "Example"}, nil
}
func (s *companyStoreStub) UpdateForEmployer(_ context.Context, _ string, input companies.UpdateInput) (companies.Company, error) {
	s.updated = input
	return companies.Company{ID: "11111111-1111-4111-8111-111111111111", Name: input.Name}, nil
}
func (s *companyStoreStub) IsFollowing(context.Context, string, string) (bool, error) {
	return false, nil
}
func (s *companyStoreStub) Follow(context.Context, string, string) error   { return nil }
func (s *companyStoreStub) Unfollow(context.Context, string, string) error { return nil }

func TestPublicCompanyEndpoints(t *testing.T) {
	store := &companyStoreStub{}
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", store)

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/companies?q=Example&page=2", nil))
	if list.Code != http.StatusOK || store.query != "Example" || store.page != 2 || !strings.Contains(list.Body.String(), `"is_verified":true`) {
		t.Fatalf("unexpected company list: %d %s", list.Code, list.Body.String())
	}

	detail := httptest.NewRecorder()
	router.ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/api/v1/companies/11111111-1111-4111-8111-111111111111", nil))
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), `"name":"Example"`) || strings.Contains(detail.Body.String(), `"jobs"`) {
		t.Fatalf("unexpected company detail: %d %s", detail.Code, detail.Body.String())
	}

	jobs := httptest.NewRecorder()
	router.ServeHTTP(jobs, httptest.NewRequest(http.MethodGet, "/api/v1/companies/11111111-1111-4111-8111-111111111111/jobs?page=1&page_size=20", nil))
	if jobs.Code != http.StatusOK || !strings.Contains(jobs.Body.String(), `"title":"Developer"`) || !strings.Contains(jobs.Body.String(), `"total_pages":1`) {
		t.Fatalf("unexpected company jobs: %d %s", jobs.Code, jobs.Body.String())
	}
}

func TestEmployerCompanyUpdateValidatesAndUsesOwnershipRoute(t *testing.T) {
	store := &companyStoreStub{}
	authService := &authStub{session: auth.Session{User: auth.User{ID: "employer", Role: auth.RoleEmployer, Status: auth.StatusActive}, CSRFToken: "csrf"}}
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", store, authService, "development")

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/employer/company", strings.NewReader(`{"name":"Updated","description":"About","website_url":"https://example.kz","logo_url":"https://example.kz/logo.svg"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost:5173")
	request.Header.Set(auth.CSRFHeaderName, "csrf")
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "token"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || store.updated.Name != "Updated" || store.updated.LogoURL != "https://example.kz/logo.svg" {
		t.Fatalf("unexpected company update: %d %s %#v", response.Code, response.Body.String(), store.updated)
	}
}

func TestFollowedCompaniesRequiresJobSeekerAndReturnsCompanies(t *testing.T) {
	store := &companyStoreStub{}
	authService := &authStub{session: auth.Session{User: auth.User{ID: "candidate", Role: auth.RoleJobSeeker, Status: auth.StatusActive}}}
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", store, authService, "development")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/companies/following", nil)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "token"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "\"name\":\"Example\"") {
		t.Fatalf("unexpected followed companies response: %d %s", response.Code, response.Body.String())
	}
}

func TestEmployerCompanyUpdateRejectsUnsafeLogoURL(t *testing.T) {
	_, fields := validateCompanyRequest(companyRequest{Name: "Example", LogoURL: "http://example.kz/logo.svg"})
	if fields["logo_url"] == "" {
		t.Fatal("expected insecure logo URL to be rejected")
	}
}

var _ companies.Store = (*companyStoreStub)(nil)
