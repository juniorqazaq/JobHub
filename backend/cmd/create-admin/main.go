package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"jobhub-ai/backend/internal/auth"
	"jobhub-ai/backend/internal/config"
	"jobhub-ai/backend/internal/database"
)

func main() {
	fullName := flag.String("full-name", "", "admin full name")
	email := flag.String("email", "", "admin email")
	passwordEnv := flag.String("password-env", "ADMIN_PASSWORD", "environment variable containing the admin password")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger, *fullName, *email, *passwordEnv); err != nil {
		logger.Error("admin provisioning failed", "error", err)
		os.Exit(1)
	}
	logger.Info("admin user provisioned", "email", *email)
}

func run(logger *slog.Logger, fullName, email, passwordEnv string) error {
	password := os.Getenv(passwordEnv)
	if password == "" {
		return fmt.Errorf("set %s with the admin password; it will not be printed or stored in source", passwordEnv)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := database.Connect(ctx, cfg.DatabaseURL, cfg.PGXQueryExecMode)
	if err != nil {
		return err
	}
	defer pool.Close()
	_, err = auth.NewService(pool).CreateAdmin(ctx, auth.AdminInput{FullName: fullName, Email: email, Password: password})
	return err
}
