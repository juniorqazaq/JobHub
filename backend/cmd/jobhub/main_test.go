package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixtureDryRunDoesNotEvenReadDatabaseEnvironment(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	fixture := filepath.Join(dir, "jobs.json")
	if err := os.WriteFile(cfg, []byte(`{"boards":[{"board_token":"fixture","display_name":"Fixture"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture, []byte(`{"jobs":[{"id":42,"title":"Engineer","absolute_url":"https://careers.example/42?token=SECRET"}],"meta":{"total":1}}`), 0600); err != nil {
		t.Fatal(err)
	}
	var out, logs bytes.Buffer
	env := func(k string) string {
		if k != "APP_ENV" {
			t.Fatalf("dry-run requested credential/config %s", k)
		}
		return "development"
	}
	if err := run([]string{"collect", "--config", cfg, "--dry-run", "--fixture", fixture}, &out, &logs, env); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"network_requests": 0`) || !strings.Contains(out.String(), `"application_method": "external"`) || strings.Contains(out.String()+logs.String(), "SECRET") {
		t.Fatalf("invalid output: %s %s", out.String(), logs.String())
	}
	// Without fixture, unapproved boards must be rejected before any request.
	if err := run([]string{"collect", "--config", cfg, "--dry-run"}, &out, &logs, env); err == nil {
		t.Fatal("unapproved live collection allowed")
	}
	if err := run([]string{"collect", "--config", cfg, "--dry-run", "--fixture", fixture}, &out, &logs, func(string) string { return "production" }); err == nil {
		t.Fatal("production allowed")
	}
}
func TestArgumentErrorsDoNotPrintSecrets(t *testing.T) {
	var out, logs bytes.Buffer
	err := run([]string{"collect", "--token=SECRET"}, &out, &logs, os.Getenv)
	if err == nil || strings.Contains(err.Error()+out.String()+logs.String(), "SECRET") {
		t.Fatal("unsafe argument error")
	}
}
