package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"jobhub-ai/backend/internal/config"
	"jobhub-ai/backend/internal/database"
	"jobhub-ai/backend/internal/locations"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	raw := os.Getenv("CITY_BACKFILL_DATABASE_URL")
	if err := config.ValidateLocalPOCDatabase(os.Getenv("APP_ENV"), raw); err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, raw, "cache_statement")
	if err != nil {
		return fmt.Errorf("connect city backfill database: %w", err)
	}
	defer pool.Close()
	stats, err := locations.BackfillImportedJobs(ctx, pool)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(stats)
}
