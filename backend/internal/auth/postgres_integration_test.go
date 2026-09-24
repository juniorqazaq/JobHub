package auth

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"jobhub-ai/backend/internal/database"
)

func TestAuthServiceIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := database.Connect(ctx, databaseURL, "cache_statement")
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	defer pool.Close()
	service := NewService(pool)
	suffix := strings.ReplaceAll(time.Now().UTC().Format("150405.000000000"), ".", "")
	email := "employer-" + suffix + "@example.com"

	session, token, err := service.Register(ctx, RegisterInput{
		FullName: "Employer Owner", Email: email, Password: "verysecure1", Role: RoleEmployer, CompanyName: "Integration LLP",
	})
	if err != nil {
		t.Fatalf("register employer: %v", err)
	}
	if session.Company == nil || session.Company.Name != "Integration LLP" || token == "" || session.CSRFToken == "" {
		t.Fatalf("unexpected session: %#v token=%q", session, token)
	}
	var membershipCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM jobhub.company_memberships WHERE user_id = $1`, session.User.ID).Scan(&membershipCount); err != nil {
		t.Fatalf("count membership: %v", err)
	}
	if membershipCount != 1 {
		t.Fatalf("expected one membership, got %d", membershipCount)
	}
	if _, _, err := service.Register(ctx, RegisterInput{FullName: "Duplicate", Email: email, Password: "verysecure1", Role: RoleJobSeeker}); err != ErrDuplicateEmail {
		t.Fatalf("expected duplicate email, got %v", err)
	}
	if _, _, err := service.Login(ctx, LoginInput{Email: email, Password: "wrong"}); err != ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	me, err := service.Authenticate(ctx, token)
	if err != nil || me.User.Email != email {
		t.Fatalf("authenticate: session=%#v err=%v", me, err)
	}
	if err := service.ValidateCSRF(ctx, token, session.CSRFToken); err != nil {
		t.Fatalf("validate csrf: %v", err)
	}
	if err := service.ValidateCSRF(ctx, token, "bad"); err != ErrForbidden {
		t.Fatalf("expected csrf forbidden, got %v", err)
	}
	if err := service.Logout(ctx, token); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := service.Authenticate(ctx, token); err != ErrInvalidSession {
		t.Fatalf("expected revoked session invalid, got %v", err)
	}
}
