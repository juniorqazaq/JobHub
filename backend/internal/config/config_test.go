package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/jobhub?sslmode=disable")
	t.Setenv("DATABASE_MIGRATION_URL", "")
	t.Setenv("PORT", "8080")
	t.Setenv("APP_ENV", "test")
	t.Setenv("PGX_QUERY_EXEC_MODE", "cache_statement")
	t.Setenv("FRONTEND_ORIGIN", "http://localhost:5173")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseMigrationURL != cfg.DatabaseURL {
		t.Fatal("migration URL should fall back to DATABASE_URL")
	}
	for _, tc := range []struct{ key, value string }{
		{"DATABASE_URL", ""},
		{"DATABASE_MIGRATION_URL", "not-a-url"},
		{"PORT", "70000"},
		{"APP_ENV", "invalid"},
		{"PGX_QUERY_EXEC_MODE", "invalid"},
		{"FRONTEND_ORIGIN", "*"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatal("expected invalid configuration to fail")
			}
		})
	}
}

func TestLoadStagingConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://postgres.project:password@pooler.example.com:6543/postgres?sslmode=require")
	t.Setenv("DATABASE_MIGRATION_URL", "postgresql://postgres.project:password@pooler.example.com:5432/postgres?sslmode=require")
	t.Setenv("APP_ENV", "staging")
	t.Setenv("PGX_QUERY_EXEC_MODE", "exec")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Environment != "staging" || cfg.PGXQueryExecMode != "exec" {
		t.Fatalf("unexpected staging config: %+v", cfg)
	}
}

func TestLoadRejectsPreparedStatementCacheForTransactionPooler(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://postgres.project:password@pooler.example.com:6543/postgres?sslmode=require")
	t.Setenv("DATABASE_MIGRATION_URL", "postgresql://postgres.project:password@pooler.example.com:5432/postgres?sslmode=require")
	t.Setenv("PGX_QUERY_EXEC_MODE", "cache_statement")

	if _, err := Load(); err == nil {
		t.Fatal("expected transaction pooler with statement cache to fail")
	}
}

func TestLoadJoobleEnforcesPOCBudget(t *testing.T) {
	t.Setenv("JOOBLE_API_KEY", "test-key")
	t.Setenv("JOOBLE_API_BASE_URL", "https://kz.jooble.org/api")
	t.Setenv("JOOBLE_MAX_REQUESTS", "50")
	t.Setenv("JOOBLE_RESULTS_PER_PAGE", "10")
	t.Setenv("JOOBLE_SEARCHES", "Go|Казахстан;бухгалтер|Казахстан")
	cfg, err := LoadJooble()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxRequests != 50 || len(cfg.Searches) != 2 {
		t.Fatalf("unexpected Jooble config: %+v", cfg)
	}
	t.Setenv("JOOBLE_MAX_REQUESTS", "51")
	if _, err := LoadJooble(); err == nil {
		t.Fatal("expected request budget above 50 to fail")
	}
}
