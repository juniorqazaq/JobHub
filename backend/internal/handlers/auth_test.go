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
	"time"

	"jobhub-ai/backend/internal/auth"
)

type authStub struct {
	session   auth.Session
	token     string
	err       error
	csrfErr   error
	loggedOut bool
}

func (s *authStub) Register(context.Context, auth.RegisterInput) (auth.Session, string, error) {
	return s.session, s.token, s.err
}
func (s *authStub) Login(context.Context, auth.LoginInput) (auth.Session, string, error) {
	return s.session, s.token, s.err
}
func (s *authStub) Authenticate(context.Context, string) (auth.Session, error) {
	return s.session, s.err
}
func (s *authStub) ValidateCSRF(context.Context, string, string) error { return s.csrfErr }
func (s *authStub) Logout(context.Context, string) error {
	s.loggedOut = true
	return s.err
}
func (s *authStub) CreateAdmin(context.Context, auth.AdminInput) (auth.User, error) {
	return auth.User{}, nil
}

func TestAuthRegisterSetsHttpOnlyCookie(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	stub := &authStub{token: "opaque-token", session: auth.Session{
		User:      auth.User{ID: "user-id", FullName: "Aigerim", Email: "a@example.com", Role: auth.RoleJobSeeker, Status: auth.StatusActive},
		CSRFToken: "csrf", ExpiresAt: now.Add(time.Hour),
	}}
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", stub, "development")
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"full_name":"Aigerim","email":"a@example.com","password":"verysecure1","role":"job_seeker"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:5173")
	router.ServeHTTP(res, req)
	if res.Code != http.StatusCreated || !strings.Contains(res.Body.String(), `"csrf_token":"csrf"`) {
		t.Fatalf("unexpected response: %d %s", res.Code, res.Body.String())
	}
	cookie := res.Result().Cookies()[0]
	if cookie.Name != auth.SessionCookieName || !cookie.HttpOnly || cookie.Value != "opaque-token" {
		t.Fatalf("unexpected cookie: %#v", cookie)
	}
}

func TestAuthRejectsAdminRegistrationAndDuplicateEmail(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"admin", auth.ErrForbidden, http.StatusForbidden},
		{"duplicate", auth.ErrDuplicateEmail, http.StatusConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", &authStub{err: tc.err}, "development")
			res := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"full_name":"Admin","email":"a@example.com","password":"verysecure1","role":"admin"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "http://localhost:5173")
			router.ServeHTTP(res, req)
			if res.Code != tc.want {
				t.Fatalf("expected %d, got %d %s", tc.want, res.Code, res.Body.String())
			}
		})
	}
}

func TestAuthMeAndLogout(t *testing.T) {
	stub := &authStub{session: auth.Session{User: auth.User{ID: "u", Role: auth.RoleEmployer, Status: auth.StatusActive}, CSRFToken: "csrf", ExpiresAt: time.Now().Add(time.Hour)}}
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", stub, "development")

	me := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "token"})
	router.ServeHTTP(me, req)
	if me.Code != http.StatusOK {
		t.Fatalf("me failed: %d %s", me.Code, me.Body.String())
	}

	logout := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set(auth.CSRFHeaderName, "csrf")
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "token"})
	router.ServeHTTP(logout, req)
	if logout.Code != http.StatusNoContent || !stub.loggedOut {
		t.Fatalf("logout failed: %d", logout.Code)
	}
}

func TestAuthLogoutRejectsMissingCSRF(t *testing.T) {
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", &authStub{csrfErr: auth.ErrForbidden}, "development")
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "token"})
	router.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected csrf failure, got %d", res.Code)
	}
}

func TestAuthMeWithoutSessionReturns401(t *testing.T) {
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", &authStub{err: auth.ErrInvalidSession}, "development")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestAuthCrossOriginMutationRejected(t *testing.T) {
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", &authStub{}, "development")
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{}`))
	req.Header.Set("Origin", "https://evil.example")
	router.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden origin, got %d", res.Code)
	}
}

func TestProtectedRoleEndpointRejectsWrongRole(t *testing.T) {
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", &authStub{session: auth.Session{User: auth.User{Role: auth.RoleJobSeeker}}}, "development")
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/employer/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "token"})
	router.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", res.Code)
	}
}

func TestAuthLoginInvalidCredentialsGeneric(t *testing.T) {
	router := NewRouter(checker{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", &authStub{err: auth.ErrInvalidCredentials}, "development")
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"a@example.com","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:5173")
	router.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized || !strings.Contains(res.Body.String(), "INVALID_CREDENTIALS") || strings.Contains(res.Body.String(), "not found") {
		t.Fatalf("unexpected response: %d %s", res.Code, res.Body.String())
	}
}

var _ auth.Service = (*authStub)(nil)
var _ = errors.Is
