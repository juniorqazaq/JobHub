package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestValidateTarget(t *testing.T) {
	for _, tc := range []struct {
		name string
		url  string
		ok   bool
	}{
		{"direct postgres", "postgresql://user:secret@example.test:5432/postgres?sslmode=require", true},
		{"session postgres", "postgres://user:secret@example.test:5432/postgres?sslmode=verify-full", true},
		{"transaction pooler", "postgresql://user:secret@example.test:6543/postgres?sslmode=require", false},
		{"insecure staging", "postgresql://user:secret@example.test:5432/postgres?sslmode=disable", false},
		{"not postgres", "https://example.test/database", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTarget(tc.url)
			if (err == nil) != tc.ok {
				t.Fatalf("validateTarget() error=%v, want ok=%t", err, tc.ok)
			}
		})
	}
}

func TestRunRequiresStagingEnvironmentAndMigrationURL(t *testing.T) {
	getenv := func(values map[string]string) func(string) string {
		return func(key string) string { return values[key] }
	}
	if err := run("version", t.TempDir(), getenv(map[string]string{
		"APP_ENV":                "production",
		"DATABASE_MIGRATION_URL": "postgresql://user:secret@example.test:5432/postgres?sslmode=require",
	})); err == nil {
		t.Fatal("expected production environment to be rejected")
	}
	if err := run("version", t.TempDir(), getenv(map[string]string{
		"APP_ENV": "staging",
	})); err == nil {
		t.Fatal("expected missing migration URL to be rejected")
	}
}

func TestMigrationVersionsAndPending(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"000001_initialize.up.sql",
		"000001_initialize.down.sql",
		"000008_add_candidate_birth_date.up.sql",
		"README.md",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	versions, err := migrationVersions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(versions, []uint{1, 8}) {
		t.Fatalf("versions=%v", versions)
	}
	if pending := pendingVersions(versions, 1); !reflect.DeepEqual(pending, []uint{8}) {
		t.Fatalf("pending=%v", pending)
	}
}

func TestSanitizeError(t *testing.T) {
	raw := "postgresql://user:super-secret@example.test:5432/postgres"
	message := sanitizeError(assertionError(raw), raw)
	if message != "connection to <redacted database URL> failed" {
		t.Fatalf("unexpected sanitized message %q", message)
	}
}

type assertionError string

func (e assertionError) Error() string { return "connection to " + string(e) + " failed" }
