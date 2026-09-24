package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"jobhub-ai/backend/internal/jobs"
)

type sourceDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	URL          string `json:"url,omitempty"`
	UpstreamName string `json:"upstream_name,omitempty"`
}

type companyDTO struct {
	ID       *string `json:"id"`
	Name     string  `json:"name"`
	Verified bool    `json:"verified"`
}

type applicationDTO struct {
	Method string `json:"method"`
	CTAURL string `json:"cta_url,omitempty"`
}

type jobDTO struct {
	ID                  string         `json:"id"`
	Title               string         `json:"title"`
	Company             companyDTO     `json:"company"`
	Location            string         `json:"location"`
	EmploymentType      string         `json:"employment_type,omitempty"`
	Category            string         `json:"category,omitempty"`
	Responsibilities    string         `json:"responsibilities,omitempty"`
	Requirements        string         `json:"requirements,omitempty"`
	NiceToHave          string         `json:"nice_to_have,omitempty"`
	Skills              []string       `json:"skills"`
	WorkMode            string         `json:"work_mode,omitempty"`
	ExperienceLevel     string         `json:"experience_level,omitempty"`
	Summary             string         `json:"summary"`
	DescriptionKind     string         `json:"description_kind"`
	SalaryRaw           string         `json:"salary_raw,omitempty"`
	SalaryMin           *float64       `json:"salary_min,omitempty"`
	SalaryMax           *float64       `json:"salary_max,omitempty"`
	SalaryCurrency      string         `json:"salary_currency,omitempty"`
	SalaryPeriod        string         `json:"salary_period,omitempty"`
	SalaryVisible       bool           `json:"salary_visible"`
	Benefits            []string       `json:"benefits"`
	PublicationStatus   string         `json:"publication_status,omitempty"`
	ModerationStatus    string         `json:"moderation_status,omitempty"`
	PostedAt            time.Time      `json:"posted_at"`
	FirstSeenAt         time.Time      `json:"first_seen_at"`
	LastSeenAt          time.Time      `json:"last_seen_at"`
	LastSyncedAt        time.Time      `json:"last_synced_at"`
	ExternalPublishedAt *time.Time     `json:"external_published_at,omitempty"`
	ExternalUpdatedAt   *time.Time     `json:"external_updated_at,omitempty"`
	ExternalExpiresAt   *time.Time     `json:"external_expires_at,omitempty"`
	ExpiresAt           *time.Time     `json:"expires_at,omitempty"`
	PublishedAt         *time.Time     `json:"published_at,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	Source              sourceDTO      `json:"source"`
	Application         applicationDTO `json:"application"`
}

type jobSearchDTO struct {
	Items      []jobDTO `json:"items"`
	Page       int      `json:"page"`
	PageSize   int      `json:"page_size"`
	Total      int64    `json:"total"`
	TotalPages int      `json:"total_pages"`
}

func ListJobs(reader jobs.Reader, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := positiveInt(c, "page", 1, 1, 100000)
		if !ok {
			return
		}
		pageSize, ok := positiveInt(c, "page_size", 20, 1, 100)
		if !ok {
			return
		}
		sort := strings.TrimSpace(c.DefaultQuery("sort", "newest"))
		if sort != "newest" && sort != "oldest" {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "sort is invalid"}})
			return
		}
		result, err := reader.Search(c.Request.Context(), jobs.SearchParams{
			Query: strings.TrimSpace(c.Query("q")), Location: strings.TrimSpace(c.Query("location")), Sort: sort, Page: page, PageSize: pageSize,
		})
		if err != nil {
			logger.Error("list jobs failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "JOBS_UNAVAILABLE", "message": "Jobs are temporarily unavailable"}})
			return
		}
		items := make([]jobDTO, 0, len(result.Items))
		for _, item := range result.Items {
			items = append(items, mapPublicJob(item))
		}
		totalPages := 0
		if result.Total > 0 {
			totalPages = int((result.Total + int64(pageSize) - 1) / int64(pageSize))
		}
		c.JSON(http.StatusOK, jobSearchDTO{Items: items, Page: page, PageSize: pageSize, Total: result.Total, TotalPages: totalPages})
	}
}

func GetJob(reader jobs.Reader, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		job, err := reader.Get(c.Request.Context(), c.Param("id"))
		if errors.Is(err, jobs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "JOB_NOT_FOUND", "message": "Job not found"}})
			return
		}
		if err != nil {
			logger.Error("get job failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "JOBS_UNAVAILABLE", "message": "Job is temporarily unavailable"}})
			return
		}
		c.JSON(http.StatusOK, mapPublicJob(job))
	}
}

func positiveInt(c *gin.Context, name string, fallback, min, max int) (int, bool) {
	raw := c.Query(name)
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < min || value > max {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": name + " is invalid"}})
		return 0, false
	}
	return value, true
}

func mapJob(job jobs.Job) jobDTO {
	postedAt := job.FirstSeenAt
	if job.PublishedAt != nil {
		postedAt = *job.PublishedAt
	} else if job.ExternalPublishedAt != nil {
		postedAt = *job.ExternalPublishedAt
	}
	var companyID *string
	if job.CompanyID != "" {
		companyID = &job.CompanyID
	}
	return jobDTO{
		ID: job.ID, Title: job.Title, Company: companyDTO{ID: companyID, Name: job.CompanyName, Verified: false},
		Location: job.Location, EmploymentType: job.EmploymentType, Category: job.Category,
		Responsibilities: job.Responsibilities, Requirements: job.Requirements, NiceToHave: job.NiceToHave,
		Skills: job.Skills, WorkMode: job.WorkMode, ExperienceLevel: job.ExperienceLevel, Summary: job.Description,
		DescriptionKind: job.DescriptionKind, SalaryRaw: job.SalaryRaw, PostedAt: postedAt,
		SalaryMin: job.SalaryMin, SalaryMax: job.SalaryMax, SalaryCurrency: job.SalaryCurrency,
		SalaryPeriod: job.SalaryPeriod, SalaryVisible: job.SalaryVisible, Benefits: job.Benefits,
		PublicationStatus: job.PublicationStatus, ModerationStatus: job.ModerationStatus,
		FirstSeenAt: job.FirstSeenAt, LastSeenAt: job.LastSeenAt, LastSyncedAt: job.LastSyncedAt,
		ExternalPublishedAt: job.ExternalPublishedAt, ExternalUpdatedAt: job.ExternalUpdatedAt,
		ExternalExpiresAt: job.ExternalExpiresAt, ExpiresAt: job.ExpiresAt, PublishedAt: job.PublishedAt,
		CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt,
		Source:      sourceDTO{ID: job.Source, Name: job.SourceName, Type: sourceType(job.Source), URL: job.SourceURL, UpstreamName: job.UpstreamSourceName},
		Application: applicationDTO{Method: job.ApplicationMethod, CTAURL: job.ApplyURL},
	}
}

func mapPublicJob(job jobs.Job) jobDTO {
	result := mapJob(job)
	if job.Source == "jobhub" && !job.SalaryVisible {
		result.SalaryRaw = ""
		result.SalaryMin = nil
		result.SalaryMax = nil
		result.SalaryCurrency = ""
		result.SalaryPeriod = ""
	}
	return result
}

func sourceType(source string) string {
	if source == "jobhub" {
		return "native"
	}
	return "external"
}
