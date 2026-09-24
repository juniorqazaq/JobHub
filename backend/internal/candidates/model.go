package candidates

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrInvalidState      = errors.New("invalid state")
	ErrJobUnavailable    = errors.New("job unavailable")
	ErrImportedJob       = errors.New("imported job")
	ErrResumeRequired    = errors.New("resume required")
	ErrResumeUnavailable = errors.New("resume unavailable")
)

type Profile struct {
	UserID                   string           `json:"user_id"`
	FullName                 string           `json:"full_name"`
	PhotoURL                 string           `json:"photo_url,omitempty"`
	City                     string           `json:"city,omitempty"`
	BirthYear                *int             `json:"birth_year,omitempty"`
	Phone                    string           `json:"phone,omitempty"`
	About                    string           `json:"about,omitempty"`
	CurrentPosition          string           `json:"current_position,omitempty"`
	DesiredPosition          string           `json:"desired_position,omitempty"`
	YearsExperience          *float64         `json:"years_experience,omitempty"`
	ExperienceLevel          string           `json:"experience_level,omitempty"`
	Skills                   []string         `json:"skills"`
	Certifications           []string         `json:"certifications"`
	Languages                []Language       `json:"languages"`
	DesiredSalary            *float64         `json:"desired_salary,omitempty"`
	Currency                 string           `json:"currency,omitempty"`
	SalaryPeriod             string           `json:"salary_period,omitempty"`
	PreferredLocations       []string         `json:"preferred_locations"`
	PreferredEmploymentTypes []string         `json:"preferred_employment_types"`
	PreferredWorkModes       []string         `json:"preferred_work_modes"`
	PreferredCategories      []string         `json:"preferred_categories"`
	PreferredRoles           []string         `json:"preferred_roles"`
	SearchStatus             string           `json:"search_status"`
	GitHub                   string           `json:"github,omitempty"`
	LinkedIn                 string           `json:"linkedin,omitempty"`
	Portfolio                string           `json:"portfolio,omitempty"`
	Website                  string           `json:"website,omitempty"`
	AllowEmployerContact     bool             `json:"allow_employer_contact"`
	ShowProfileToEmployers   bool             `json:"show_profile_to_employers"`
	ShowSalaryExpectations   bool             `json:"show_salary_expectations"`
	WorkExperience           []WorkExperience `json:"work_experience"`
	Education                []Education      `json:"education"`
	Resume                   *Resume          `json:"resume,omitempty"`
	Completion               Completion       `json:"completion"`
}

type WorkExperience struct {
	ID             string   `json:"id,omitempty"`
	Company        string   `json:"company"`
	Position       string   `json:"position"`
	EmploymentType string   `json:"employment_type"`
	StartDate      string   `json:"start_date"`
	EndDate        string   `json:"end_date,omitempty"`
	IsCurrent      bool     `json:"is_current"`
	Description    string   `json:"description,omitempty"`
	Achievements   string   `json:"achievements,omitempty"`
	Skills         []string `json:"skills"`
}

type Education struct {
	ID             string `json:"id,omitempty"`
	Institution    string `json:"institution"`
	Degree         string `json:"degree"`
	FieldOfStudy   string `json:"field_of_study"`
	StartYear      int    `json:"start_year"`
	GraduationYear *int   `json:"graduation_year,omitempty"`
	Description    string `json:"description,omitempty"`
}

type Language struct {
	ID          string `json:"id,omitempty"`
	Language    string `json:"language"`
	Proficiency string `json:"proficiency"`
}

type Completion struct {
	Percentage int      `json:"percentage"`
	Missing    []string `json:"missing"`
}

type Resume struct {
	ID               string    `json:"id"`
	OriginalFilename string    `json:"original_filename"`
	ContentType      string    `json:"content_type"`
	SizeBytes        int64     `json:"size_bytes"`
	UploadedAt       time.Time `json:"uploaded_at"`
	Status           string    `json:"status"`
	storageKey       string
}

type SavedJob struct {
	JobID       string    `json:"job_id"`
	SavedAt     time.Time `json:"saved_at"`
	Available   bool      `json:"available"`
	Title       string    `json:"title,omitempty"`
	CompanyName string    `json:"company_name,omitempty"`
	Location    string    `json:"location,omitempty"`
	Source      string    `json:"source,omitempty"`
	SourceName  string    `json:"source_name,omitempty"`
	Application string    `json:"application_method,omitempty"`
}

type Application struct {
	ID                string    `json:"id"`
	JobID             string    `json:"job_id"`
	CompanyID         string    `json:"company_id"`
	CandidateID       string    `json:"candidate_id,omitempty"`
	CandidateName     string    `json:"candidate_name,omitempty"`
	CandidatePosition string    `json:"candidate_position,omitempty"`
	JobTitle          string    `json:"job_title,omitempty"`
	CompanyName       string    `json:"company_name,omitempty"`
	JobAvailable      bool      `json:"job_available"`
	Status            string    `json:"status"`
	Message           string    `json:"message,omitempty"`
	Resume            Resume    `json:"resume"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	LatestStatusAt    time.Time `json:"latest_status_at"`
}

type ApplicationEvent struct {
	PreviousStatus string    `json:"previous_status,omitempty"`
	NewStatus      string    `json:"new_status"`
	CreatedAt      time.Time `json:"created_at"`
}

type Store interface {
	GetProfile(context.Context, string) (Profile, error)
	UpdateProfile(context.Context, string, Profile) (Profile, error)
	GetActiveResume(context.Context, string) (Resume, error)
	ReplaceResume(context.Context, string, Resume) (Resume, *Resume, error)
	DeleteResume(context.Context, string) (*Resume, bool, error)
	SaveJob(context.Context, string, string) error
	UnsaveJob(context.Context, string, string) error
	ListSavedJobs(context.Context, string) ([]SavedJob, error)
	CreateApplication(context.Context, string, string, string, string) (Application, error)
	ListCandidateApplications(context.Context, string) ([]Application, error)
	WithdrawApplication(context.Context, string, string) (Application, error)
	ListEmployerApplications(context.Context, string, string) ([]Application, error)
	UpdateEmployerApplication(context.Context, string, string, string) (Application, error)
	GetEmployerApplicationResume(context.Context, string, string) (Resume, error)
}

type FileStore interface {
	Put(context.Context, string, io.Reader) error
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}
