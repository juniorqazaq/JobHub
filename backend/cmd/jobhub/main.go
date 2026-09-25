package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"jobhub-ai/backend/internal/config"
	"jobhub-ai/backend/internal/database"
	"jobhub-ai/backend/internal/ingestion"
	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/providers"
	"jobhub-ai/backend/internal/providers/airastana"
	"jobhub-ai/backend/internal/providers/greenhouse"
	"jobhub-ai/backend/internal/providers/kcell"
	"jobhub-ai/backend/internal/providers/staticcareers"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr, os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string, out, logs io.Writer, getenv func(string) string) error {
	if len(args) == 0 || args[0] != "collect" {
		return errors.New("usage: jobhub collect --config <server-config.json> --source <provider:source> --dry-run")
	}
	flags := flag.NewFlagSet("collect", flag.ContinueOnError)
	flags.SetOutput(io.Discard) // Parse errors must not echo sensitive arguments.
	path := flags.String("config", "", "server-side source config")
	source := flags.String("source", "", "source ID (default: all configured boards)")
	dry := flags.Bool("dry-run", false, "collect without DB access")
	ingest := flags.Bool("ingest", false, "isolated development ingestion")
	health := flags.Bool("health", false, "read persisted development source health only")
	fixture := flags.String("fixture", "", "offline JSON fixture; dry-run only")
	sample := flags.Int("sample", 5, "sample rows, 0–10")
	if flags.Parse(args[1:]) != nil || flags.NArg() != 0 {
		return errors.New("invalid collect arguments")
	}
	modes := 0
	for _, mode := range []bool{*dry, *ingest, *health} {
		if mode {
			modes++
		}
	}
	if modes != 1 || *path == "" || *sample < 0 || *sample > 10 || (*fixture != "" && !*dry) {
		return errors.New("select exactly one of --dry-run, --ingest, --health; fixtures require --dry-run")
	}
	if env := getenv("APP_ENV"); env == "production" || env == "staging" {
		return errors.New("ATS POC is disabled in staging and production")
	}
	cfg, err := config.LoadATS(*path)
	if err != nil {
		return err
	}
	type selection struct {
		source, provider, display, board, listingURL, detailPrefix, company string
		authorized                                                          bool
	}
	selected := []selection{}
	for _, board := range cfg.Boards {
		if *source == "" || *source == "greenhouse:"+board.BoardToken {
			selected = append(selected, selection{"greenhouse:" + board.BoardToken, "greenhouse", board.DisplayName, board.BoardToken, "", "", "", board.AuthorizedForPOC})
		}
	}
	for _, career := range cfg.CareerSources {
		id := career.Provider + ":careers"
		if *source == "" || *source == id {
			selected = append(selected, selection{id, career.Provider, career.DisplayName, "", career.ListingURL, career.DetailPrefix, career.Company, career.AuthorizedForPOC})
		}
	}
	if len(selected) == 0 || (*fixture != "" && len(selected) != 1) {
		return errors.New("unknown source or fixture requires one source")
	}
	if !*health && *fixture == "" {
		for _, selectedSource := range selected {
			if !selectedSource.authorized {
				return errors.New("source has no recorded POC authorization")
			}
		}
	}
	budget, err := greenhouse.NewBudget(cfg.MaxRequests, cfg.MaxJobs)
	if err != nil {
		return err
	}
	var client *http.Client
	if *fixture != "" {
		f, err := os.Open(*fixture)
		if err != nil {
			return errors.New("cannot read fixture")
		}
		body, err := io.ReadAll(io.LimitReader(f, (5<<20)+1))
		f.Close()
		if err != nil || len(body) > 5<<20 {
			return errors.New("invalid fixture size")
		}
		client = &http.Client{Transport: fixtureTransport(body)}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	var store *jobs.PostgresStore
	if !*dry {
		raw := getenv("ATS_DEV_DATABASE_URL")
		if err := config.ValidateATSDevelopmentDatabase(getenv("APP_ENV"), raw); err != nil {
			return err
		}
		pool, err := database.Connect(ctx, raw, "cache_statement")
		if err != nil {
			return errors.New("isolated development database connection failed")
		}
		defer pool.Close()
		store = jobs.NewPostgresStore(pool)
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	for n, selectedSource := range selected {
		id := selectedSource.source
		if *health {
			h, err := store.SourceHealth(ctx, id)
			if err != nil {
				return err
			}
			if err := encoder.Encode(h); err != nil {
				return errors.New("report output failed")
			}
			continue
		}
		if n > 0 && *fixture == "" {
			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
				return errors.New("collection deadline exceeded")
			}
		}
		var p providers.VacancyProvider
		switch selectedSource.provider {
		case "greenhouse":
			p, err = greenhouse.NewClient(selectedSource.board, budget, client)
		case "kcell":
			p, err = kcell.NewClient(min(cfg.MaxRequests, 5), cfg.MaxJobs, client)
		case "airastana":
			p, err = airastana.NewClient(min(cfg.MaxRequests, 10), cfg.MaxJobs, client)
		case "halyk", "technodom", "kolesa":
			p, err = staticcareers.New(staticcareers.Config{Source: id, Company: selectedSource.company, ListingURL: selectedSource.listingURL, DetailPrefix: selectedSource.detailPrefix}, cfg.MaxRequests, cfg.MaxJobs, client)
		default:
			return errors.New("unsupported provider")
		}
		if err != nil {
			return err
		}
		var target jobs.Store
		if !*dry {
			if err := store.RegisterDevelopmentSource(ctx, id, selectedSource.provider, selectedSource.display); err != nil {
				return err
			}
			target = store
		}
		service := ingestion.NewService(target, nil, logger, 0)
		report, collectErr := service.Run(ctx, p, ingestion.RunOptions{DryRun: *dry, Sample: *sample, NoAutomaticExpiry: true})
		mode := "live"
		network := report.Stats.RequestCount
		if *fixture != "" {
			mode = "fixture"
			network = 0
		}
		var h *jobs.SourceHealth
		if !*dry {
			value, err := store.SourceHealth(ctx, id)
			if err != nil {
				return err
			}
			h = &value
		}
		if err := encoder.Encode(struct {
			Mode            string             `json:"mode"`
			NetworkRequests int                `json:"network_requests"`
			Report          ingestion.Report   `json:"report"`
			Health          *jobs.SourceHealth `json:"persisted_health,omitempty"`
		}{mode, network, report, h}); err != nil {
			return errors.New("report output failed")
		}
		if collectErr != nil {
			return collectErr
		}
	}
	return nil
}

type fixtureTransport []byte

func (f fixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(f))), Request: req}, nil
}
