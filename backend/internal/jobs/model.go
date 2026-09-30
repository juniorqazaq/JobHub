package jobs

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("job not found")
var ErrInvalidTransition = errors.New("invalid job status transition")

type ImportedJob struct {
	Source              string
	ExternalID          string
	SourceURL           string
	UpstreamSourceName  string
	CompanyNameRaw      string
	Title               string
	LocationRaw         string
	CanonicalCityID     string
	Description         string
	DescriptionKind     string
	EmploymentTypeRaw   string
	SalaryRaw           string
	Category            string
	ExternalPublishedAt *time.Time
	ExternalUpdatedAt   *time.Time
	ExternalUpdatedRaw  string
	ExternalExpiresAt   *time.Time
}

type Job struct {
	ID                  string
	Source              string
	SourceName          string
	ExternalID          string
	SourceURL           string
	UpstreamSourceName  string
	CompanyName         string
	CompanyID           string
	CompanyLogoURL      string
	CompanyVerified     bool
	Title               string
	Category            string
	Location            string
	CanonicalCityID     string
	Description         string
	Responsibilities    string
	Requirements        string
	NiceToHave          string
	Skills              []string
	WorkMode            string
	ExperienceLevel     string
	DescriptionKind     string
	EmploymentType      string
	SalaryRaw           string
	SalaryMin           *float64
	SalaryMax           *float64
	SalaryCurrency      string
	SalaryPeriod        string
	SalaryVisible       bool
	Benefits            []string
	ApplicationMethod   string
	ApplyURL            string
	PublicationStatus   string
	ModerationStatus    string
	FirstSeenAt         time.Time
	LastSeenAt          time.Time
	LastSyncedAt        time.Time
	ExternalPublishedAt *time.Time
	ExternalUpdatedAt   *time.Time
	ExternalExpiresAt   *time.Time
	ExpiresAt           *time.Time
	PublishedAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type NativeJobInput struct {
	Title            string
	Category         string
	Description      string
	Responsibilities string
	Requirements     string
	NiceToHave       string
	Skills           []string
	CanonicalCityID  string
	WorkMode         string
	EmploymentType   string
	ExperienceLevel  string
	SalaryMin        *float64
	SalaryMax        *float64
	SalaryCurrency   string
	SalaryPeriod     string
	SalaryVisible    bool
	Benefits         []string
	ExpiresAt        *time.Time
}

type SearchParams struct {
	Query           string
	City            string
	PreferredCity   string
	WorkModes       []string
	SalaryMin       *float64
	Currency        string
	ExperienceLevel string
	EmploymentType  string
	PostedAfter     *time.Time
	Sort            string
	Page            int
	PageSize        int
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

type EmployerStore interface {
	ListForEmployer(context.Context, string) ([]Job, error)
	GetForEmployer(context.Context, string, string) (Job, error)
	CreateForEmployer(context.Context, string, NativeJobInput) (Job, error)
	UpdateForEmployer(context.Context, string, string, NativeJobInput) (Job, error)
	TransitionForEmployer(context.Context, string, string, string) (Job, error)
	DeleteForEmployer(context.Context, string, string) error
}

type Store interface {
	Reader
	BeginIngestionRun(context.Context, string, int) (string, error)
	FailIngestionRun(context.Context, string, ImportStats, string, string) error
	CompleteIngestionRun(context.Context, string, string, []ImportedJob, time.Time, time.Time, ImportStats) (ImportStats, error)
}
