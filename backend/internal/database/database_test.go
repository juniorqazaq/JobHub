package database

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestPoolConfigQueryExecMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want pgx.QueryExecMode
	}{
		{"direct or session", "cache_statement", pgx.QueryExecModeCacheStatement},
		{"transaction pooler", "exec", pgx.QueryExecModeExec},
		{"simple protocol", "simple_protocol", pgx.QueryExecModeSimpleProtocol},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := PoolConfig("postgresql://user:password@localhost:5432/jobhub?sslmode=disable", tt.mode)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.ConnConfig.DefaultQueryExecMode != tt.want {
				t.Fatalf("got %v, want %v", cfg.ConnConfig.DefaultQueryExecMode, tt.want)
			}
		})
	}
}

func TestPoolConfigRejectsUnsupportedMode(t *testing.T) {
	if _, err := PoolConfig("postgresql://user:password@localhost:5432/jobhub", "unknown"); err == nil {
		t.Fatal("expected unsupported mode to fail")
	}
}
