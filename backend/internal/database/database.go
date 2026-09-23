package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

func Connect(ctx context.Context, databaseURL, queryExecMode string) (*pgxpool.Pool, error) {
	cfg, err := PoolConfig(databaseURL, queryExecMode)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("initialize database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	return pool, nil
}

func PoolConfig(databaseURL, queryExecMode string) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid database configuration")
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second

	// Supabase's transaction pooler (port 6543) does not support prepared
	// statements. "exec" keeps pgx's extended protocol while avoiding its
	// prepared-statement cache; direct and session connections can keep the
	// faster default cache_statement mode.
	switch queryExecMode {
	case "cache_statement":
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement
	case "exec":
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	case "simple_protocol":
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	default:
		return nil, fmt.Errorf("unsupported PGX query execution mode")
	}

	return cfg, nil
}
