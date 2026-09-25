package ingestion

import (
	"context"
	"log/slog"
	"net/url"
	"sort"
	"strings"
	"time"

	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/locations"
	"jobhub-ai/backend/internal/providers"
	"jobhub-ai/backend/internal/providers/jooble"
)

type JoobleSearcher interface {
	Search(context.Context, jooble.Search) ([]jooble.Vacancy, error)
	RequestCount() int
}

type Service struct {
	store    jobs.Store
	provider JoobleSearcher
	logger   *slog.Logger
	freshFor time.Duration
	now      func() time.Time
}

func NewService(store jobs.Store, provider JoobleSearcher, logger *slog.Logger, freshFor time.Duration) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{store: store, provider: provider, logger: logger, freshFor: freshFor, now: time.Now}
}

type RunOptions struct {
	DryRun            bool
	Sample            int
	SearchCount       int
	NoAutomaticExpiry bool
}
type Report struct {
	Source            string           `json:"source"`
	DryRun            bool             `json:"dry_run"`
	AttemptAt         time.Time        `json:"attempt_at"`
	Stats             jobs.ImportStats `json:"stats"`
	Malformed         int              `json:"malformed"`
	Duplicates        int              `json:"duplicates"`
	CompleteSnapshot  bool             `json:"complete_snapshot"`
	ListRequests      int              `json:"list_requests"`
	DetailRequests    int              `json:"detail_requests"`
	PagesFetched      int              `json:"pages_fetched"`
	DetailUnavailable int              `json:"detail_unavailable"`
	MaxRequests       int              `json:"max_requests"`
	RequestsUsed      int              `json:"requests_used"`
	RemainingRequests int              `json:"remaining"`
	FailureCategory   string           `json:"failure_category,omitempty"`
	Samples           []Sample         `json:"samples"`
}
type Sample struct {
	ExternalID        string     `json:"external_id"`
	Title             string     `json:"title"`
	Company           *string    `json:"company"`
	Location          *string    `json:"location"`
	CanonicalCityID   *string    `json:"canonical_city_id"`
	DescriptionKind   string     `json:"description_kind"`
	ApplicationMethod string     `json:"application_method"`
	ApplicationURL    string     `json:"application_url_redacted"`
	UpdatedAt         *time.Time `json:"external_updated_at"`
}

// Run shares validation, dedupe and atomic persistence across providers. DryRun
// never accesses the store, including run creation or source-health mutations.
func (s *Service) Run(ctx context.Context, p providers.VacancyProvider, opts RunOptions) (Report, error) {
	report := Report{Source: p.Source(), DryRun: opts.DryRun, AttemptAt: s.now().UTC(), Samples: []Sample{}}
	report.Stats.SearchCount = opts.SearchCount
	if opts.Sample < 0 || opts.Sample > 10 {
		return report, providers.Failure("INVALID_SAMPLE_LIMIT")
	}
	if !opts.DryRun {
		if s.store == nil {
			return report, providers.Failure("STORE_REQUIRED")
		}
		id, err := s.store.BeginIngestionRun(ctx, p.Source(), opts.SearchCount)
		if err != nil {
			return report, providers.Failure("DATABASE_WRITE_FAILED")
		}
		report.Stats.RunID = id
	}
	fail := func(category string) (Report, error) {
		report.FailureCategory = category
		if !opts.DryRun {
			failureCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			if err := s.store.FailIngestionRun(failureCtx, report.Stats.RunID, report.Stats, category, category); err != nil {
				report.FailureCategory = "RUN_RECORD_FAILED"
			}
		}
		s.logger.Error("vacancy collection failed", "category", report.FailureCategory, "requests", report.Stats.RequestCount)
		return report, providers.Failure(report.FailureCategory)
	}
	result, err := p.Collect(ctx)
	report.Stats.RequestCount = result.Requests
	report.Stats.FetchedCount = result.Fetched
	report.Malformed = result.Malformed
	report.Stats.SkippedCount = result.Skipped + result.Malformed
	report.CompleteSnapshot = result.Complete
	report.ListRequests = result.ListRequests
	report.DetailRequests = result.DetailRequests
	report.PagesFetched = result.PagesFetched
	report.DetailUnavailable = result.DetailUnavailable
	report.MaxRequests = result.MaxRequests
	report.RequestsUsed = result.RequestsUsed
	report.RemainingRequests = result.RemainingRequests
	if err != nil {
		report.CompleteSnapshot = false
		return fail(providers.Category(err))
	}
	unique := map[string]jobs.ImportedJob{}
	for _, item := range result.Items {
		if item.CanonicalCityID == "" {
			item.CanonicalCityID = locations.Normalize(item.LocationRaw)
		}
		parsed, err := url.Parse(item.SourceURL)
		if item.Source != p.Source() || strings.TrimSpace(item.ExternalID) == "" || strings.TrimSpace(item.Title) == "" || (item.CanonicalCityID != "" && !locations.Valid(item.CanonicalCityID)) || err != nil || parsed.Hostname() == "" || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || (item.DescriptionKind != "full" && item.DescriptionKind != "snippet") {
			report.Malformed++
			report.Stats.SkippedCount++
			report.CompleteSnapshot = false
			continue
		}
		report.Stats.NormalizedCount++
		if _, exists := unique[item.ExternalID]; exists {
			report.Duplicates++
			report.Stats.SkippedCount++
			report.CompleteSnapshot = false
		}
		unique[item.ExternalID] = item
	}
	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]jobs.ImportedJob, 0, len(ids))
	for _, id := range ids {
		item := unique[id]
		items = append(items, item)
		if len(report.Samples) < opts.Sample {
			u, _ := url.Parse(item.SourceURL)
			u.RawQuery = ""
			u.Fragment = ""
			report.Samples = append(report.Samples, Sample{ExternalID: id, Title: item.Title, Company: optional(item.CompanyNameRaw), Location: optional(item.LocationRaw), CanonicalCityID: optional(item.CanonicalCityID), DescriptionKind: item.DescriptionKind, ApplicationMethod: "external", ApplicationURL: u.String(), UpdatedAt: item.ExternalUpdatedAt})
		}
	}
	if opts.DryRun {
		return report, nil
	}
	// No absence reconciliation in this POC. This explicit development policy
	// avoids inventing closure or applying Jooble's stale TTL to ATS jobs.
	freshUntil := report.AttemptAt.Add(s.freshFor)
	if opts.NoAutomaticExpiry {
		freshUntil = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	}
	stats, err := s.store.CompleteIngestionRun(ctx, report.Stats.RunID, p.Source(), items, report.AttemptAt, freshUntil, report.Stats)
	if err != nil {
		return fail("DATABASE_WRITE_FAILED")
	}
	report.Stats = stats
	s.logger.Info("vacancy ingestion completed", "requests", stats.RequestCount, "fetched", stats.FetchedCount, "inserted", stats.InsertedCount, "updated", stats.UpdatedCount)
	return report, nil
}
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *Service) RunJooble(ctx context.Context, searches []jooble.Search) (jobs.ImportStats, error) {
	report, err := s.Run(ctx, joobleCollection{s.provider, searches}, RunOptions{SearchCount: len(searches)})
	return report.Stats, err
}

type joobleCollection struct {
	client   JoobleSearcher
	searches []jooble.Search
}

func (j joobleCollection) Source() string { return jooble.Source }
func (j joobleCollection) Collect(ctx context.Context) (providers.Result, error) {
	result := providers.Result{}
	start := j.client.RequestCount()
	for _, search := range j.searches {
		vacancies, err := j.client.Search(ctx, search)
		result.Requests = j.client.RequestCount() - start
		if err != nil {
			return result, providers.Failure("PROVIDER_REQUEST_FAILED")
		}
		result.Fetched += len(vacancies)
		for _, v := range vacancies {
			item, err := jooble.Normalize(v)
			if err != nil {
				result.Malformed++
				continue
			}
			result.Items = append(result.Items, item)
		}
	}
	return result, nil // Ranked search is never a complete snapshot.
}
