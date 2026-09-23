package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"jobhub-ai/backend/internal/config"
	"jobhub-ai/backend/internal/database"
	"jobhub-ai/backend/internal/ingestion"
	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/providers/jooble"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("Jooble import stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	appConfig, err := config.Load()
	if err != nil {
		return err
	}
	if appConfig.Environment == "production" {
		return errors.New("Jooble importing is disabled in production until API-specific provider permissions are confirmed")
	}
	providerConfig, err := config.LoadJooble()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := database.Connect(ctx, appConfig.DatabaseURL, appConfig.PGXQueryExecMode)
	if err != nil {
		return err
	}
	defer pool.Close()
	client, err := jooble.NewClient(providerConfig.BaseURL, providerConfig.APIKey, providerConfig.MaxRequests, &http.Client{Timeout: 15 * time.Second}, logger)
	if err != nil {
		return err
	}
	searches := make([]jooble.Search, 0, len(providerConfig.Searches))
	for _, configured := range providerConfig.Searches {
		searches = append(searches, jooble.Search{Keywords: configured.Keywords, Location: configured.Location, Page: 1, ResultsPerPage: providerConfig.ResultsPerPage})
	}
	service := ingestion.NewService(jobs.NewPostgresStore(pool), client, logger, providerConfig.FreshFor)
	_, err = service.RunJooble(ctx, searches)
	return err
}
