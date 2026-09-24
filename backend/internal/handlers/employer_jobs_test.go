package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jobhub-ai/backend/internal/auth"
	"jobhub-ai/backend/internal/jobs"
)

type employerStoreStub struct {
	item        jobs.Job
	err         error
	createCalls int
}

func (s *employerStoreStub) ListForEmployer(context.Context, string) ([]jobs.Job, error) {
	return []jobs.Job{s.item}, s.err
}
func (s *employerStoreStub) GetForEmployer(context.Context, string, string) (jobs.Job, error) {
	return s.item, s.err
}
func (s *employerStoreStub) CreateForEmployer(_ context.Context, _ string, input jobs.NativeJobInput) (jobs.Job, error) {
	s.createCalls++
	if s.err != nil {
		return jobs.Job{}, s.err
	}
	s.item = jobs.Job{ID: "11111111-1111-4111-8111-111111111111", Source: "jobhub", Title: input.Title, PublicationStatus: "draft", ApplicationMethod: "internal"}
	return s.item, nil
}
func (s *employerStoreStub) UpdateForEmployer(context.Context, string, string, jobs.NativeJobInput) (jobs.Job, error) {
	return s.item, s.err
}
func (s *employerStoreStub) TransitionForEmployer(_ context.Context, _, _, status string) (jobs.Job, error) {
	if s.err != nil {
		return jobs.Job{}, s.err
	}
	if s.item.PublicationStatus == "closed" && status != "closed" {
		return jobs.Job{}, jobs.ErrInvalidTransition
	}
	s.item.PublicationStatus = status
	return s.item, nil
}
func (s *employerStoreStub) DeleteForEmployer(context.Context, string, string) error { return s.err }

func TestEmployerCreatesDraftVacancy(t *testing.T) {
	store := &employerStoreStub{}
	router := employerRouter(auth.RoleEmployer, store)
	res := performEmployerMutation(router, http.MethodPost, "/api/v1/jobs", validNativeJobBody())
	if res.Code != http.StatusCreated || store.createCalls != 1 || !strings.Contains(res.Body.String(), `"publication_status":"draft"`) || !strings.Contains(res.Body.String(), `"id":"jobhub"`) {
		t.Fatalf("unexpected create response: %d %s", res.Code, res.Body.String())
	}
}

func TestJobSeekerCannotCreateVacancy(t *testing.T) {
	store := &employerStoreStub{}
	res := performEmployerMutation(employerRouter(auth.RoleJobSeeker, store), http.MethodPost, "/api/v1/jobs", validNativeJobBody())
	if res.Code != http.StatusForbidden || store.createCalls != 0 {
		t.Fatalf("expected forbidden, got %d", res.Code)
	}
}

func TestEmployerCannotMutateUnownedOrImportedVacancy(t *testing.T) {
	for _, name := range []string{"another company", "imported vacancy"} {
		t.Run(name, func(t *testing.T) {
			store := &employerStoreStub{err: jobs.ErrNotFound}
			res := performEmployerMutation(employerRouter(auth.RoleEmployer, store), http.MethodPatch, "/api/v1/jobs/11111111-1111-4111-8111-111111111111", validNativeJobBody())
			if res.Code != http.StatusNotFound {
				t.Fatalf("expected hidden resource, got %d", res.Code)
			}
		})
	}
}

func TestClosedVacancyCannotBeRepublished(t *testing.T) {
	store := &employerStoreStub{item: jobs.Job{PublicationStatus: "closed"}}
	res := performEmployerMutation(employerRouter(auth.RoleEmployer, store), http.MethodPatch, "/api/v1/jobs/11111111-1111-4111-8111-111111111111", `{"publication_status":"published"}`)
	if res.Code != http.StatusConflict || !strings.Contains(res.Body.String(), "INVALID_JOB_TRANSITION") {
		t.Fatalf("unexpected response: %d %s", res.Code, res.Body.String())
	}
}

func TestEmployerMutationRequiresCSRF(t *testing.T) {
	store := &employerStoreStub{}
	service := &authStub{csrfErr: auth.ErrForbidden, session: auth.Session{User: auth.User{ID: "11111111-1111-4111-8111-111111111111", Role: auth.RoleEmployer}}}
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", jobReaderStub{}, store, service)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(validNativeJobBody()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:5173")
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "token"})
	router.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden || store.createCalls != 0 {
		t.Fatalf("expected csrf rejection, got %d", res.Code)
	}
}

func TestNativeJobValidationPreservesStructuredSkillsAndLongPlainText(t *testing.T) {
	minimum, maximum := 100000.0, 300000.0
	request := nativeJobRequest{
		Title: " Backend Developer ", Category: "Engineering",
		Description:      strings.Repeat("Ұ", 200) + "\n<script>alert(1)</script>",
		Responsibilities: "Build APIs\nReview code", Requirements: "Production Go experience",
		Skills: []string{" Go ", "Rust", "React Native"}, Benefits: []string{" Learning budget "},
		Location: "Алматы", WorkMode: "hybrid", EmploymentType: "full_time", ExperienceLevel: "middle",
		SalaryMin: &minimum, SalaryMax: &maximum, SalaryCurrency: "usd", SalaryPeriod: "month",
	}
	input, fields := request.input()
	if len(fields) != 0 {
		t.Fatalf("expected valid input, got %#v", fields)
	}
	if input.Title != "Backend Developer" || len(input.Skills) != 3 || input.Skills[2] != "React Native" || input.SalaryCurrency != "USD" {
		t.Fatalf("normalization corrupted structured input: %#v", input)
	}
	if !strings.Contains(input.Description, "<script>") {
		t.Fatalf("plain text content was unexpectedly truncated or altered")
	}
}

func TestNativeJobValidationRejectsMalformedContentAndSalary(t *testing.T) {
	negative, smaller := -1.0, 10.0
	request := nativeJobRequest{
		Title: strings.Repeat("x", jobTitleMax+1), Category: "Engineering", Description: "A sufficiently detailed description",
		Responsibilities: "Develop APIs", Requirements: "Go experience", Skills: []string{"Go", "   "},
		Location: "Almaty", WorkMode: "hybrid", EmploymentType: "full_time", ExperienceLevel: "middle",
		SalaryMin: &negative, SalaryMax: &smaller, SalaryCurrency: "US", SalaryPeriod: "week",
	}
	_, fields := request.input()
	for _, field := range []string{"title", "skills", "salary_min", "salary_currency", "salary_period"} {
		if fields[field] == "" {
			t.Errorf("expected validation error for %s, got %#v", field, fields)
		}
	}
}

func TestNativeJobValidationRejectsInvertedSalaryRange(t *testing.T) {
	minimum, maximum := 300000.0, 100000.0
	request := nativeJobRequest{
		Title: "Backend Developer", Category: "Engineering", Description: "A sufficiently detailed description",
		Responsibilities: "Develop APIs", Requirements: "Go experience", Location: "Almaty",
		WorkMode: "hybrid", EmploymentType: "full_time", ExperienceLevel: "middle",
		SalaryMin: &minimum, SalaryMax: &maximum, SalaryCurrency: "KZT", SalaryPeriod: "month",
	}
	_, fields := request.input()
	if fields["salary_max"] == "" {
		t.Fatalf("expected inverted range error, got %#v", fields)
	}
}

func employerRouter(role string, store *employerStoreStub) http.Handler {
	service := &authStub{session: auth.Session{User: auth.User{ID: "11111111-1111-4111-8111-111111111111", Role: role, Status: auth.StatusActive}}}
	return NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", jobReaderStub{}, store, service)
}

func performEmployerMutation(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set(auth.CSRFHeaderName, "csrf")
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "token"})
	router.ServeHTTP(res, req)
	return res
}

func validNativeJobBody() string {
	return `{"title":"Backend Developer","category":"Engineering","description":"Build reliable services","responsibilities":"Develop APIs","requirements":"Go experience","location":"Almaty","work_mode":"hybrid","employment_type":"full_time","experience_level":"middle","salary_visible":true}`
}

var _ jobs.EmployerStore = (*employerStoreStub)(nil)
