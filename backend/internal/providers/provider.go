// Package providers defines the collection boundary shared by vacancy adapters.
package providers

import (
	"context"
	"errors"

	"jobhub-ai/backend/internal/jobs"
)

type Result struct {
	Items             []jobs.ImportedJob
	Requests          int
	ListRequests      int
	DetailRequests    int
	PagesFetched      int
	DetailUnavailable int
	MaxRequests       int
	RequestsUsed      int
	RemainingRequests int
	Fetched           int
	Malformed         int
	Skipped           int
	Complete          bool
}

type VacancyProvider interface {
	Source() string
	Collect(context.Context) (Result, error)
}

// Failure contains a safe category only. Never wrap transport errors or bodies:
// these can contain credential-bearing URLs or untrusted provider content.
type Failure string

func (f Failure) Error() string { return string(f) }
func Category(err error) string {
	var failure Failure
	if errors.As(err, &failure) {
		return string(failure)
	}
	return "PROVIDER_REQUEST_FAILED"
}
