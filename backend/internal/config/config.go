package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment          string
	Port                 string
	DatabaseURL          string
	DatabaseMigrationURL string
	PGXQueryExecMode     string
	FrontendOrigin       string
}

type JoobleSearch struct {
	Keywords string
	Location string
}

type JoobleConfig struct {
	APIKey         string
	BaseURL        string
	MaxRequests    int
	ResultsPerPage int
	FreshFor       time.Duration
	Searches       []JoobleSearch
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

func LoadJooble() (JoobleConfig, error) {
	maxRequests, err := strconv.Atoi(value("JOOBLE_MAX_REQUESTS", "10"))
	if err != nil || maxRequests < 1 || maxRequests > 50 {
		return JoobleConfig{}, fmt.Errorf("JOOBLE_MAX_REQUESTS must be between 1 and 50")
	}
	resultsPerPage, err := strconv.Atoi(value("JOOBLE_RESULTS_PER_PAGE", "10"))
	if err != nil || resultsPerPage < 1 || resultsPerPage > 20 {
		return JoobleConfig{}, fmt.Errorf("JOOBLE_RESULTS_PER_PAGE must be between 1 and 20")
	}
	freshFor, err := time.ParseDuration(value("JOOBLE_FRESH_FOR", "72h"))
	if err != nil || freshFor < time.Hour || freshFor > 7*24*time.Hour {
		return JoobleConfig{}, fmt.Errorf("JOOBLE_FRESH_FOR must be between 1h and 168h")
	}
	baseURL := strings.TrimRight(value("JOOBLE_API_BASE_URL", "https://kz.jooble.org/api"), "/")
	parsedURL, err := url.Parse(baseURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.RawQuery != "" || parsedURL.Fragment != "" || parsedURL.User != nil {
		return JoobleConfig{}, fmt.Errorf("JOOBLE_API_BASE_URL must be an HTTP(S) URL without credentials, query, or fragment")
	}
	apiKey := strings.TrimSpace(os.Getenv("JOOBLE_API_KEY"))
	if apiKey == "" {
		return JoobleConfig{}, fmt.Errorf("JOOBLE_API_KEY is required for the Jooble importer")
	}
	searches, err := parseJoobleSearches(value("JOOBLE_SEARCHES", "программист|Казахстан;бухгалтер|Казахстан;менеджер по продажам|Казахстан"))
	if err != nil {
		return JoobleConfig{}, err
	}
	if len(searches) > maxRequests {
		return JoobleConfig{}, fmt.Errorf("JOOBLE_SEARCHES requires more requests than JOOBLE_MAX_REQUESTS allows")
	}
	return JoobleConfig{
		APIKey:         apiKey,
		BaseURL:        baseURL,
		MaxRequests:    maxRequests,
		ResultsPerPage: resultsPerPage,
		FreshFor:       freshFor,
		Searches:       searches,
	}, nil
}

func parseJoobleSearches(raw string) ([]JoobleSearch, error) {
	parts := strings.Split(raw, ";")
	if len(parts) == 0 || len(parts) > 5 {
		return nil, fmt.Errorf("JOOBLE_SEARCHES must contain between 1 and 5 searches")
	}
	searches := make([]JoobleSearch, 0, len(parts))
	for _, part := range parts {
		fields := strings.Split(part, "|")
		if len(fields) != 2 {
			return nil, fmt.Errorf("JOOBLE_SEARCHES entries must use keywords|location")
		}
		keywords := strings.TrimSpace(fields[0])
		location := strings.TrimSpace(fields[1])
		if keywords == "" || location == "" {
			return nil, fmt.Errorf("JOOBLE_SEARCHES keywords and location must not be blank")
		}
		searches = append(searches, JoobleSearch{Keywords: keywords, Location: location})
	}
	return searches, nil
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
