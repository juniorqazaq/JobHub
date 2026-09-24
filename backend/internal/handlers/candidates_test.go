package handlers

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jobhub-ai/backend/internal/auth"
	"jobhub-ai/backend/internal/candidates"
)

func TestCandidateRoutesEnforceRoles(t *testing.T) {
	for _, tc := range []struct {
		name, role, method, path string
		want                     int
	}{
		{"employer cannot read profile", auth.RoleEmployer, http.MethodGet, "/api/v1/profile", http.StatusForbidden},
		{"candidate cannot list employer applicants", auth.RoleJobSeeker, http.MethodGet, "/api/v1/employer/jobs/11111111-1111-4111-8111-111111111111/applications", http.StatusForbidden},
		{"admin cannot read candidate profile", auth.RoleAdmin, http.MethodGet, "/api/v1/profile", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router, _ := candidateRouter(tc.role, nil)
			res := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "token"})
			router.ServeHTTP(res, req)
			if res.Code != tc.want {
				t.Fatalf("want %d got %d: %s", tc.want, res.Code, res.Body.String())
			}
		})
	}
}

func TestCandidateCannotApplyToImportedJob(t *testing.T) {
	router, _ := candidateRouter(auth.RoleJobSeeker, candidates.ErrImportedJob)
	res := candidateMutation(router, http.MethodPost, "/api/v1/jobs/11111111-1111-4111-8111-111111111111/applications", `{"resume_id":"22222222-2222-4222-8222-222222222222"}`)
	if res.Code != http.StatusBadRequest || !bytes.Contains(res.Body.Bytes(), []byte("EXTERNAL_JOB")) {
		t.Fatalf("unexpected response: %d %s", res.Code, res.Body.String())
	}
}

func TestCandidateMutationRequiresCSRF(t *testing.T) {
	store := &candidateStoreStub{}
	files := candidateFilesStub{}
	service := candidates.NewService(store, files)
	authService := &authStub{csrfErr: auth.ErrForbidden, session: auth.Session{User: auth.User{ID: "11111111-1111-4111-8111-111111111111", Role: auth.RoleJobSeeker, Status: auth.StatusActive}}}
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", jobReaderStub{}, authService, service)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/jobs/22222222-2222-4222-8222-222222222222/saved", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "token"})
	router.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden || store.saveCalls != 0 {
		t.Fatalf("expected csrf rejection, got %d", res.Code)
	}
}

func TestSavedJobPreflightAllowsPut(t *testing.T) {
	router, _ := candidateRouter(auth.RoleJobSeeker, nil)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/jobs/22222222-2222-4222-8222-222222222222/saved", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPut)
	router.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent || !strings.Contains(res.Header().Get("Access-Control-Allow-Methods"), http.MethodPut) {
		t.Fatalf("expected PUT in CORS methods, got %d %q", res.Code, res.Header().Get("Access-Control-Allow-Methods"))
	}
}

func TestValidateProfileNormalizesKazakhstanPhone(t *testing.T) {
	profile, fields := validateProfile(profileRequest{
		FullName:     "Candidate",
		SearchStatus: "actively_looking",
		Phone:        "8 (700) 123-45-67",
	})
	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if profile.Phone != "+77001234567" {
		t.Fatalf("expected canonical phone, got %q", profile.Phone)
	}
}

func TestValidateProfilePreservesInternationalPhone(t *testing.T) {
	profile, fields := validateProfile(profileRequest{
		FullName:     "Candidate",
		SearchStatus: "actively_looking",
		Phone:        "+44 20 7946 0958",
	})
	if len(fields) != 0 || profile.Phone != "+442079460958" {
		t.Fatalf("expected canonical international phone, got %q fields=%#v", profile.Phone, fields)
	}
}

func TestValidateProfileRejectsInvalidPhoneBirthDateAndURL(t *testing.T) {
	_, fields := validateProfile(profileRequest{
		FullName:     "Candidate",
		SearchStatus: "actively_looking",
		Phone:        "+7 123",
		BirthDate:    "2023-02-29",
		Website:      "https://user:password@example.com/private",
	})
	for _, field := range []string{"phone", "birth_date", "website"} {
		if _, ok := fields[field]; !ok {
			t.Fatalf("expected %s validation error, got %#v", field, fields)
		}
	}
}

func TestValidateProfileAcceptsLeapDay(t *testing.T) {
	profile, fields := validateProfile(profileRequest{
		FullName:     "Candidate",
		SearchStatus: "actively_looking",
		BirthDate:    "2000-02-29",
		GitHub:       "https://github.com/candidate",
	})
	if len(fields) != 0 || profile.BirthDate != "2000-02-29" {
		t.Fatalf("expected valid leap day, got %#v fields=%#v", profile, fields)
	}
}

func TestValidateProfileStoresCanonicalCitiesAndPreservesUnknownLegacyLocation(t *testing.T) {
	profile, fields := validateProfile(profileRequest{
		FullName: "Candidate", SearchStatus: "actively_looking", City: "old value", CityID: "almaty",
		PreferredLocations: []string{"Remote across Kazakhstan", "Astana"}, PreferredCityIDs: []string{"astana", "shymkent"},
	})
	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if profile.City != "Almaty" || profile.CityID != "almaty" || len(profile.PreferredLocations) != 3 || profile.PreferredLocations[0] != "Remote across Kazakhstan" || profile.PreferredLocations[1] != "Astana" || profile.PreferredLocations[2] != "Shymkent" {
		t.Fatalf("canonical or legacy locations were not preserved: %#v", profile)
	}
}

func TestValidateProfileRejectsInvalidOrDuplicateCanonicalCities(t *testing.T) {
	_, fields := validateProfile(profileRequest{
		FullName: "Candidate", SearchStatus: "actively_looking", CityID: "unknown", PreferredCityIDs: []string{"astana", "astana"},
	})
	if fields["city_id"] == "" || fields["preferred_city_ids"] == "" {
		t.Fatalf("expected canonical city validation errors, got %#v", fields)
	}
}

func candidateRouter(role string, storeErr error) (http.Handler, *candidateStoreStub) {
	store := &candidateStoreStub{err: storeErr}
	service := candidates.NewService(store, candidateFilesStub{})
	authService := &authStub{session: auth.Session{User: auth.User{ID: "11111111-1111-4111-8111-111111111111", Role: role, Status: auth.StatusActive}}}
	return NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", jobReaderStub{}, authService, service), store
}
func candidateMutation(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set(auth.CSRFHeaderName, "csrf")
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "token"})
	router.ServeHTTP(res, req)
	return res
}

type candidateFilesStub struct{}

func (candidateFilesStub) Put(context.Context, string, io.Reader) error { return nil }
func (candidateFilesStub) Open(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}
func (candidateFilesStub) Delete(context.Context, string) error { return nil }

type candidateStoreStub struct {
	err       error
	saveCalls int
}

func (s *candidateStoreStub) GetProfile(context.Context, string) (candidates.Profile, error) {
	return candidates.Profile{FullName: "Candidate"}, s.err
}
func (s *candidateStoreStub) UpdateProfile(context.Context, string, candidates.Profile) (candidates.Profile, error) {
	return candidates.Profile{}, s.err
}
func (s *candidateStoreStub) GetActiveResume(context.Context, string) (candidates.Resume, error) {
	return candidates.Resume{}, s.err
}
func (s *candidateStoreStub) ReplaceResume(context.Context, string, candidates.Resume) (candidates.Resume, *candidates.Resume, error) {
	return candidates.Resume{}, nil, s.err
}
func (s *candidateStoreStub) DeleteResume(context.Context, string) (*candidates.Resume, bool, error) {
	return nil, false, s.err
}
func (s *candidateStoreStub) SaveJob(context.Context, string, string) error {
	s.saveCalls++
	return s.err
}
func (s *candidateStoreStub) UnsaveJob(context.Context, string, string) error { return s.err }
func (s *candidateStoreStub) ListSavedJobs(context.Context, string) ([]candidates.SavedJob, error) {
	return nil, s.err
}
func (s *candidateStoreStub) CreateApplication(context.Context, string, string, string, string) (candidates.Application, error) {
	return candidates.Application{}, s.err
}
func (s *candidateStoreStub) ListCandidateApplications(context.Context, string) ([]candidates.Application, error) {
	return nil, s.err
}
func (s *candidateStoreStub) WithdrawApplication(context.Context, string, string) (candidates.Application, error) {
	return candidates.Application{}, s.err
}
func (s *candidateStoreStub) ListEmployerApplications(context.Context, string, string) ([]candidates.Application, error) {
	return nil, s.err
}
func (s *candidateStoreStub) UpdateEmployerApplication(context.Context, string, string, string) (candidates.Application, error) {
	return candidates.Application{}, s.err
}
func (s *candidateStoreStub) GetEmployerApplicationResume(context.Context, string, string) (candidates.Resume, error) {
	return candidates.Resume{}, s.err
}
