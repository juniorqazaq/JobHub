package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestATSDevelopmentDatabaseBoundary(t *testing.T) {
	good := "postgres://poc@127.0.0.1:55479/jobhub_ats_poc_test?sslmode=disable"
	if err := ValidateATSDevelopmentDatabase("development", good); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{strings.Replace(good, "127.0.0.1", "remote.example", 1), strings.Replace(good, "jobhub_ats_poc_test", "production", 1), good + "&host=remote.example", good + "&hostaddr=1.1.1.1", good + "&service=remote", good + "&sslmode=%ZZ", "not a URL"} {
		if ValidateATSDevelopmentDatabase("development", raw) == nil {
			t.Fatalf("accepted unsafe database %s", raw)
		}
	}
	for _, env := range []string{"", "staging", "production", "test"} {
		if ValidateATSDevelopmentDatabase(env, good) == nil {
			t.Fatal("accepted environment", env)
		}
	}
}
func TestATSConfigLimits(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		valid bool
	}{
		{`{"boards":[{"board_token":"fixture","display_name":"Fixture"}]}`, true},
		{`{"boards":[{"board_token":"../secret","display_name":"Fixture"}]}`, false},
		{`{"boards":[{"board_token":"fixture","display_name":"Fixture"}],"max_requests":13}`, false},
		{`{"boards":[{"board_token":"fixture","display_name":"Fixture"}],"max_jobs":201}`, false},
		{`{"boards":[{"board_token":"fixture","display_name":"Fixture"}],"api_key":"SECRET"}`, false},
		{`{"boards":[{"board_token":"fixture","display_name":"Fixture"},{"board_token":"fixture","display_name":"Other"}]}`, false},
		{`{"boards":[{"board_token":"a","display_name":"A"},{"board_token":"b","display_name":"B"},{"board_token":"c","display_name":"C"}]}`, false},
		{`{"boards":[{"board_token":"a","display_name":"A"},{"board_token":"b","display_name":"B"},{"board_token":"c","display_name":"C","test_board":true}]}`, true},
		{`{"career_sources":[{"provider":"kcell","display_name":"Kcell","authorized_for_poc":true},{"provider":"airastana","display_name":"Air Astana","authorized_for_poc":true}]}`, true},
		{`{"career_sources":[{"provider":"lever","display_name":"Lever"}]}`, false},
	} {
		p := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(p, []byte(tc.raw), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := LoadATS(p)
		if (err == nil) != tc.valid {
			t.Fatalf("valid=%v err=%v", tc.valid, err)
		}
	}
}
