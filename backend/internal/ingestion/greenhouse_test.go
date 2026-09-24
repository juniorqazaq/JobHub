package ingestion

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/providers"
)

type collection struct {
	result providers.Result
	err    error
}

func (collection) Source() string                                      { return "greenhouse:fixture" }
func (c collection) Collect(context.Context) (providers.Result, error) { return c.result, c.err }

type panicStore struct{ jobs.Store }

func (*panicStore) BeginIngestionRun(context.Context, string, int) (string, error) {
	panic("dry-run called DB")
}
func (*panicStore) FailIngestionRun(context.Context, string, jobs.ImportStats, string, string) error {
	panic("dry-run called DB")
}
func (*panicStore) CompleteIngestionRun(context.Context, string, string, []jobs.ImportedJob, time.Time, time.Time, jobs.ImportStats) (jobs.ImportStats, error) {
	panic("unused")
}
func TestDryRunNoStoreCallsDedupeAndSanitizedFailure(t *testing.T) {
	var logs bytes.Buffer
	svc := NewService(&panicStore{}, nil, slog.New(slog.NewTextHandler(&logs, nil)), 0)
	item := jobs.ImportedJob{Source: "greenhouse:fixture", ExternalID: "42", Title: "Engineer", SourceURL: "https://careers.example/42?token=SECRET", DescriptionKind: "full"}
	p := collection{result: providers.Result{Items: []jobs.ImportedJob{item, item}, Fetched: 3, Requests: 1, Malformed: 1, Complete: true}}
	r, err := svc.Run(context.Background(), p, RunOptions{DryRun: true, Sample: 5})
	if err != nil || r.Stats.NormalizedCount != 2 || r.Stats.SkippedCount != 2 || r.Duplicates != 1 || r.CompleteSnapshot || r.Stats.RunID != "" || len(r.Samples) != 1 || r.Samples[0].ApplicationURL != "https://careers.example/42" || r.Samples[0].Company != nil {
		t.Fatalf("%+v %v", r, err)
	}
	r, err = svc.Run(context.Background(), collection{err: fmt.Errorf("SECRET")}, RunOptions{DryRun: true})
	if err == nil || r.FailureCategory != "PROVIDER_REQUEST_FAILED" || bytes.Contains(logs.Bytes(), []byte("SECRET")) {
		t.Fatalf("unsafe failure: %s", logs.String())
	}
}
func TestMalformedSourceCannotCrossProviderBoundary(t *testing.T) {
	svc := NewService(nil, nil, nil, 0)
	r, err := svc.Run(context.Background(), collection{result: providers.Result{Complete: true, Items: []jobs.ImportedJob{{Source: "jobhub", ExternalID: "42", Title: "Wrong source", SourceURL: "https://example.com", DescriptionKind: "full"}}}}, RunOptions{DryRun: true, Sample: 5})
	if err != nil || r.Malformed != 1 || r.CompleteSnapshot || len(r.Samples) != 0 {
		t.Fatalf("%+v %v", r, err)
	}
}
