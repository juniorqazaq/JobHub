package companies

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("company not found")

type Company struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	WebsiteURL    string    `json:"website_url,omitempty"`
	LogoURL       string    `json:"logo_url,omitempty"`
	Industry      string    `json:"industry,omitempty"`
	City          string    `json:"city,omitempty"`
	Verified      bool      `json:"is_verified"`
	OpenJobsCount int64     `json:"open_jobs_count"`
	FollowerCount int64     `json:"follower_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Vacancy struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Location       string     `json:"location,omitempty"`
	CityID         string     `json:"city_id,omitempty"`
	WorkMode       string     `json:"work_mode,omitempty"`
	EmploymentType string     `json:"employment_type,omitempty"`
	SalaryMin      *float64   `json:"salary_min,omitempty"`
	SalaryMax      *float64   `json:"salary_max,omitempty"`
	SalaryCurrency string     `json:"salary_currency,omitempty"`
	SalaryPeriod   string     `json:"salary_period,omitempty"`
	SalaryVisible  bool       `json:"salary_visible"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	Source         string     `json:"source"`
	SourceName     string     `json:"source_name"`
}

type SearchResult struct {
	Items []Company
	Total int64
}

type VacancySearchResult struct {
	Items []Vacancy
	Total int64
}

type UpdateInput struct {
	Name        string
	Description string
	WebsiteURL  string
	LogoURL     string
	Industry    string
	City        string
}

type Store interface {
	Search(context.Context, string, int, int) (SearchResult, error)
	ListFollowed(context.Context, string) ([]Company, error)
	GetPublic(context.Context, string) (Company, error)
	ListPublicJobs(context.Context, string, int, int) (VacancySearchResult, error)
	GetForEmployer(context.Context, string) (Company, error)
	UpdateForEmployer(context.Context, string, UpdateInput) (Company, error)
	IsFollowing(context.Context, string, string) (bool, error)
	Follow(context.Context, string, string) error
	Unfollow(context.Context, string, string) error
}
