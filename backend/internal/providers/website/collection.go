package website

import (
	"context"
	"strings"

	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/providers"
)

type Collection struct {
	SourceName        string
	Items             []jobs.ImportedJob
	Requests, Skipped int
}

func (c Collection) Source() string { return c.SourceName }
func (c Collection) Collect(context.Context) (providers.Result, error) {
	if !strings.HasPrefix(c.SourceName, "website:") && !strings.HasPrefix(c.SourceName, "greenhouse:") {
		return providers.Result{}, providers.Failure("INVALID_SOURCE")
	}
	items := append([]jobs.ImportedJob(nil), c.Items...)
	return providers.Result{Items: items, Requests: c.Requests, Fetched: len(items) + c.Skipped, Skipped: c.Skipped, Complete: false}, nil
}
