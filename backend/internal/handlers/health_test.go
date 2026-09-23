package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

type checker struct{ err error }

func (s checker) Check(context.Context) error { return s.err }
func TestHealth(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{"healthy", nil, 200, `"database":"up"`},
		{"unavailable", errors.New("private connection details"), 503, `"code":"DATABASE_UNAVAILABLE"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter(checker{tc.err}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173")
			req := httptest.NewRequest("GET", "/api/v1/health", nil)
			req.Header.Set("Origin", "http://localhost:5173")
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != tc.status || !strings.Contains(res.Body.String(), tc.body) {
				t.Fatalf("unexpected response: %d %s", res.Code, res.Body.String())
			}
			if strings.Contains(res.Body.String(), "private") {
				t.Fatal("internal details exposed")
			}
			if res.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
				t.Fatal("missing CORS header")
			}
		})
	}
}
