package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
)

var migrationName = regexp.MustCompile(`^(\d+)_.*\.up\.sql$`)

func main() {
	action := flag.String("action", "version", "version or up")
	path := flag.String("path", "./migrations", "directory containing numbered migrations")
	flag.Parse()

	if err := run(*action, *path, os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, "migration command failed:", sanitizeError(err, os.Getenv("DATABASE_MIGRATION_URL")))
		os.Exit(1)
	}
}

func run(action, path string, getenv func(string) string) error {
	if action != "version" && action != "up" {
		return fmt.Errorf("unsupported action %q; use version or up", action)
	}
	if getenv("APP_ENV") != "staging" {
		return errors.New("APP_ENV must be staging; production and other environments are not allowed by this command")
	}
	databaseURL := strings.TrimSpace(getenv("DATABASE_MIGRATION_URL"))
	if databaseURL == "" {
		return errors.New("DATABASE_MIGRATION_URL is required")
	}
	if err := validateTarget(databaseURL); err != nil {
		return err
	}

	versions, err := migrationVersions(path)
	if err != nil {
		return err
	}
	current, dirty, err := readDatabaseVersion(databaseURL)
	if err != nil {
		return err
	}
	fmt.Printf("environment=staging current=%d dirty=%t\n", current, dirty)
	if dirty {
		return errors.New("database migration state is dirty; refusing to continue")
	}
	pending := pendingVersions(versions, current)
	if len(pending) == 0 {
		fmt.Println("pending=none")
	} else {
		fmt.Printf("pending=%s\n", joinVersions(pending))
	}
	if action == "version" || len(pending) == 0 {
		return nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return errors.New("cannot resolve the migration directory")
	}
	sourceURL := (&url.URL{Scheme: "file", Path: absPath}).String()
	runner, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return fmt.Errorf("cannot initialize the migration runner: %w", err)
	}
	defer func() { _, _ = runner.Close() }()

	if err := runner.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		failedVersion, failedDirty, versionErr := migrationVersion(runner)
		if versionErr == nil {
			fmt.Fprintf(os.Stderr, "result current=%d dirty=%t\n", failedVersion, failedDirty)
		}
		return fmt.Errorf("up migration failed: %w", err)
	}
	current, dirty, err = migrationVersion(runner)
	if err != nil {
		return fmt.Errorf("migration completed but final version could not be read: %w", err)
	}
	fmt.Printf("result current=%d dirty=%t\n", current, dirty)
	if dirty {
		return errors.New("migration completed with a dirty state")
	}
	return nil
}

func readDatabaseVersion(databaseURL string) (uint, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return 0, false, errors.New("cannot connect to the staging migration database")
	}
	defer func() { _ = connection.Close(context.Background()) }()
	var version uint
	var dirty bool
	if err := connection.QueryRow(ctx, `SELECT version, dirty FROM public.schema_migrations LIMIT 1`).Scan(&version, &dirty); err != nil {
		return 0, false, errors.New("cannot read public.schema_migrations")
	}
	return version, dirty, nil
}

func validateTarget(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return errors.New("DATABASE_MIGRATION_URL must be a PostgreSQL connection URL")
	}
	if parsed.Port() == "6543" {
		return errors.New("transaction pooler port 6543 is not allowed; use a direct or session connection on port 5432")
	}
	if strings.EqualFold(parsed.Query().Get("sslmode"), "disable") {
		return errors.New("sslmode=disable is not allowed for staging migrations")
	}
	return nil
}

func migrationVersions(path string) ([]uint, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, errors.New("cannot read the migration directory")
	}
	versions := make([]uint, 0, len(entries)/2)
	seen := make(map[uint]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		match := migrationName.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}
		value, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil || value == 0 {
			return nil, fmt.Errorf("invalid migration filename %q", entry.Name())
		}
		version := uint(value)
		if seen[version] {
			return nil, fmt.Errorf("duplicate up migration version %d", version)
		}
		seen[version] = true
		versions = append(versions, version)
	}
	if len(versions) == 0 {
		return nil, errors.New("no numbered up migrations found")
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
	return versions, nil
}

func migrationVersion(runner *migrate.Migrate) (uint, bool, error) {
	version, dirty, err := runner.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("cannot read migration version: %w", err)
	}
	return version, dirty, nil
}

func pendingVersions(versions []uint, current uint) []uint {
	pending := make([]uint, 0, len(versions))
	for _, version := range versions {
		if version > current {
			pending = append(pending, version)
		}
	}
	return pending
}

func joinVersions(versions []uint) string {
	values := make([]string, len(versions))
	for i, version := range versions {
		values[i] = strconv.FormatUint(uint64(version), 10)
	}
	return strings.Join(values, ",")
}

func sanitizeError(err error, rawURL string) string {
	message := err.Error()
	if rawURL != "" {
		message = strings.ReplaceAll(message, rawURL, "<redacted database URL>")
	}
	if parsed, parseErr := url.Parse(rawURL); parseErr == nil && parsed.User != nil {
		if password, ok := parsed.User.Password(); ok && password != "" {
			message = strings.ReplaceAll(message, password, "<redacted>")
		}
	}
	return message
}
