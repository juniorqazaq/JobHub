package jobs

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("job not found")

type ImportedJob struct {
	Source             string
	ExternalID         string
	SourceURL          string
	UpstreamSourceName string
	CompanyNameRaw     string
	Title              string
	LocationRaw        string
	Description        string
	DescriptionKind    string
	EmploymentTypeRaw  string
	SalaryRaw          string
	ExternalUpdatedAt  *time.Time
	ExternalUpdatedRaw string
}

type Job struct {
	ID                  string
	Source              string
	SourceName          string
	ExternalID          string
	SourceURL           string
	UpstreamSourceName  string
	CompanyName         string
	Title               string
	Location            string
	Description         string
	DescriptionKind     string
	EmploymentType      string
	SalaryRaw           string
	ApplicationMethod   string
	ApplyURL            string
	FirstSeenAt         time.Time
	LastSeenAt          time.Time
	LastSyncedAt        time.Time
	ExternalPublishedAt *time.Time
	ExternalUpdatedAt   *time.Time
	ExternalExpiresAt   *time.Time
}

type SearchParams struct {
	Query    string
	Location string
	Sort     string
	Page     int
	PageSize int
}

type SearchResult struct {
	Items []Job
	Total int64
}

type ImportStats struct {
	RunID           string
	SearchCount     int
	RequestCount    int
	FetchedCount    int
	NormalizedCount int
	InsertedCount   int
	UpdatedCount    int
	SkippedCount    int
}

type Reader interface {
	Search(context.Context, SearchParams) (SearchResult, error)
	Get(context.Context, string) (Job, error)
}

type Store interface {
	Reader
	BeginIngestionRun(context.Context, string, int) (string, error)
	FailIngestionRun(context.Context, string, ImportStats, string, string) error
	CompleteIngestionRun(context.Context, string, string, []ImportedJob, time.Time, time.Time, ImportStats) (ImportStats, error)
}
