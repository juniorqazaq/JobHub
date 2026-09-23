package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

type Config struct {
	Environment          string
	Port                 string
	DatabaseURL          string
	DatabaseMigrationURL string
	PGXQueryExecMode     string
	FrontendOrigin       string
}

func Load() (Config, error) {
	c := Config{
		Environment:          value("APP_ENV", "development"),
		Port:                 value("PORT", "8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		DatabaseMigrationURL: os.Getenv("DATABASE_MIGRATION_URL"),
		PGXQueryExecMode:     value("PGX_QUERY_EXEC_MODE", "cache_statement"),
		FrontendOrigin:       value("FRONTEND_ORIGIN", "http://localhost:5173"),
	}
	if c.DatabaseMigrationURL == "" {
		c.DatabaseMigrationURL = c.DatabaseURL
	}
	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be between 1 and 65535")
	}
	if !validDatabaseURL(c.DatabaseURL) {
		return Config{}, fmt.Errorf("DATABASE_URL must be a PostgreSQL connection URL")
	}
	if !validDatabaseURL(c.DatabaseMigrationURL) {
		return Config{}, fmt.Errorf("DATABASE_MIGRATION_URL must be a PostgreSQL connection URL")
	}
	origin, err := url.Parse(c.FrontendOrigin)
	if err != nil || origin.Host == "" || (origin.Scheme != "http" && origin.Scheme != "https") || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" || origin.User != nil {
		return Config{}, fmt.Errorf("FRONTEND_ORIGIN must be an HTTP(S) origin without a path")
	}
	if c.Environment != "development" && c.Environment != "test" && c.Environment != "staging" && c.Environment != "production" {
		return Config{}, fmt.Errorf("APP_ENV must be development, test, staging, or production")
	}
	if c.PGXQueryExecMode != "cache_statement" && c.PGXQueryExecMode != "exec" && c.PGXQueryExecMode != "simple_protocol" {
		return Config{}, fmt.Errorf("PGX_QUERY_EXEC_MODE must be cache_statement, exec, or simple_protocol")
	}
	databaseURL, _ := url.Parse(c.DatabaseURL)
	if databaseURL.Port() == "6543" && c.PGXQueryExecMode == "cache_statement" {
		return Config{}, fmt.Errorf("PGX_QUERY_EXEC_MODE must be exec or simple_protocol for a transaction pooler on port 6543")
	}
	return c, nil
}

func validDatabaseURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Host != "" && (u.Scheme == "postgres" || u.Scheme == "postgresql")
}
func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
