package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"jobhub-ai/backend/internal/auth"
	"jobhub-ai/backend/internal/candidates"
	"jobhub-ai/backend/internal/config"
	"jobhub-ai/backend/internal/database"
	"jobhub-ai/backend/internal/handlers"
	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/services"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("application stopped", "error", err)
		os.Exit(1)
	}
}
func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startup, cancel := context.WithTimeout(ctx, 10*time.Second)
	pool, err := database.Connect(startup, cfg.DatabaseURL, cfg.PGXQueryExecMode)
	cancel()
	if err != nil {
		return err
	}
	defer pool.Close()
	jobStore := jobs.NewPostgresStore(pool, cfg.Environment == "production")
	authService := auth.NewService(pool)
	resumeFiles, err := candidates.NewLocalFileStore(cfg.ResumeStorageDir)
	if err != nil {
		return err
	}
	candidateService := candidates.NewService(candidates.NewPostgresStore(pool), resumeFiles)
	router := handlers.NewRouter(services.NewHealthService(pool), logger, cfg.FrontendOrigin, jobStore, authService, candidateService, cfg.Environment)
	if err := router.SetTrustedProxies(nil); err != nil {
		return err
	}
	// Profile updates can span several remote Postgres round trips. Keep the API and
	// browser timeouts aligned so a committed request is not reported as failed.
	server := &http.Server{Addr: ":" + cfg.Port, Handler: router, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	failures := make(chan error, 1)
	go func() { failures <- server.ListenAndServe() }()
	logger.Info("API started", "port", cfg.Port, "environment", cfg.Environment)
	select {
	case err := <-failures:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
	}
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	return server.Shutdown(shutdown)
}
