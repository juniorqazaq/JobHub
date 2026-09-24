package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"jobhub-ai/backend/internal/auth"
	"jobhub-ai/backend/internal/jobs"
)

type nativeJobRequest struct {
	Title             string     `json:"title"`
	Category          string     `json:"category"`
	Description       string     `json:"description"`
	Responsibilities  string     `json:"responsibilities"`
	Requirements      string     `json:"requirements"`
	NiceToHave        string     `json:"nice_to_have"`
	Skills            []string   `json:"skills"`
	Location          string     `json:"location"`
	WorkMode          string     `json:"work_mode"`
	EmploymentType    string     `json:"employment_type"`
	ExperienceLevel   string     `json:"experience_level"`
	SalaryMin         *float64   `json:"salary_min"`
	SalaryMax         *float64   `json:"salary_max"`
	SalaryCurrency    string     `json:"salary_currency"`
	SalaryPeriod      string     `json:"salary_period"`
	SalaryVisible     *bool      `json:"salary_visible"`
	Benefits          []string   `json:"benefits"`
	ExpiresAt         *time.Time `json:"expires_at"`
	PublicationStatus string     `json:"publication_status"`
}

func RegisterEmployerJobRoutes(router *gin.Engine, store jobs.EmployerStore, authService auth.Service, logger *slog.Logger, origin string) {
	protected := []gin.HandlerFunc{RequireRole(authService, auth.RoleEmployer)}
	mutation := []gin.HandlerFunc{RequireTrustedOrigin(origin), RequireRole(authService, auth.RoleEmployer), RequireCSRF(authService)}
	router.GET("/api/v1/employer/jobs", append(protected, listEmployerJobs(store, logger))...)
	router.GET("/api/v1/employer/jobs/:id", append(protected, getEmployerJob(store, logger))...)
	router.POST("/api/v1/jobs", append(mutation, createEmployerJob(store, logger))...)
	router.PATCH("/api/v1/jobs/:id", append(mutation, updateEmployerJob(store, logger))...)
	router.DELETE("/api/v1/jobs/:id", append(mutation, deleteEmployerJob(store, logger))...)
}

func listEmployerJobs(store jobs.EmployerStore, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := store.ListForEmployer(c.Request.Context(), currentUser(c).ID)
		if err != nil {
			writeEmployerJobError(c, logger, err)
			return
		}
		response := make([]jobDTO, 0, len(items))
		for _, item := range items {
			response = append(response, mapJob(item))
		}
		c.JSON(http.StatusOK, gin.H{"items": response})
	}
}

func getEmployerJob(store jobs.EmployerStore, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := store.GetForEmployer(c.Request.Context(), currentUser(c).ID, c.Param("id"))
		if err != nil {
			writeEmployerJobError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, mapJob(item))
	}
}

func createEmployerJob(store jobs.EmployerStore, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request nativeJobRequest
		if !decodeJSON(c, &request) {
			return
		}
		input, fields := request.input()
		if len(fields) > 0 {
			writeJobValidation(c, fields)
			return
		}
		item, err := store.CreateForEmployer(c.Request.Context(), currentUser(c).ID, input)
		if err != nil {
			writeEmployerJobError(c, logger, err)
			return
		}
		c.JSON(http.StatusCreated, mapJob(item))
	}
}

func updateEmployerJob(store jobs.EmployerStore, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request nativeJobRequest
		if !decodeJSON(c, &request) {
			return
		}
		if request.PublicationStatus != "" {
			if !validStatus(request.PublicationStatus) {
				writeJobValidation(c, map[string]string{"publication_status": "invalid status"})
				return
			}
			item, err := store.TransitionForEmployer(c.Request.Context(), currentUser(c).ID, c.Param("id"), request.PublicationStatus)
			if err != nil {
				writeEmployerJobError(c, logger, err)
				return
			}
			c.JSON(http.StatusOK, mapJob(item))
			return
		}
		input, fields := request.input()
		if len(fields) > 0 {
			writeJobValidation(c, fields)
			return
		}
		item, err := store.UpdateForEmployer(c.Request.Context(), currentUser(c).ID, c.Param("id"), input)
		if err != nil {
			writeEmployerJobError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, mapJob(item))
	}
}

func deleteEmployerJob(store jobs.EmployerStore, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := store.DeleteForEmployer(c.Request.Context(), currentUser(c).ID, c.Param("id")); err != nil {
			writeEmployerJobError(c, logger, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func (r nativeJobRequest) input() (jobs.NativeJobInput, map[string]string) {
	fields := map[string]string{}
	required := map[string]string{"title": r.Title, "category": r.Category, "description": r.Description, "responsibilities": r.Responsibilities, "requirements": r.Requirements, "location": r.Location, "work_mode": r.WorkMode, "employment_type": r.EmploymentType, "experience_level": r.ExperienceLevel}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			fields[field] = "required"
		}
	}
	if !oneOf(r.WorkMode, "on_site", "hybrid", "remote") {
		fields["work_mode"] = "invalid value"
	}
	if !oneOf(r.EmploymentType, "full_time", "part_time", "contract", "temporary", "internship") {
		fields["employment_type"] = "invalid value"
	}
	if !oneOf(r.ExperienceLevel, "no_experience", "junior", "middle", "senior", "lead") {
		fields["experience_level"] = "invalid value"
	}
	if r.SalaryMin != nil && *r.SalaryMin < 0 {
		fields["salary_min"] = "must be non-negative"
	}
	if r.SalaryMax != nil && *r.SalaryMax < 0 {
		fields["salary_max"] = "must be non-negative"
	}
	if r.SalaryMin != nil && r.SalaryMax != nil && *r.SalaryMax < *r.SalaryMin {
		fields["salary_max"] = "must be greater than or equal to salary_min"
	}
	if r.ExpiresAt != nil && !r.ExpiresAt.After(time.Now()) {
		fields["expires_at"] = "must be in the future"
	}
	salaryVisible := true
	if r.SalaryVisible != nil {
		salaryVisible = *r.SalaryVisible
	}
	return jobs.NativeJobInput{Title: strings.TrimSpace(r.Title), Category: strings.TrimSpace(r.Category), Description: strings.TrimSpace(r.Description), Responsibilities: strings.TrimSpace(r.Responsibilities), Requirements: strings.TrimSpace(r.Requirements), NiceToHave: strings.TrimSpace(r.NiceToHave), Skills: cleanList(r.Skills), Location: strings.TrimSpace(r.Location), WorkMode: r.WorkMode, EmploymentType: r.EmploymentType, ExperienceLevel: r.ExperienceLevel, SalaryMin: r.SalaryMin, SalaryMax: r.SalaryMax, SalaryCurrency: strings.ToUpper(strings.TrimSpace(r.SalaryCurrency)), SalaryPeriod: strings.TrimSpace(r.SalaryPeriod), SalaryVisible: salaryVisible, Benefits: cleanList(r.Benefits), ExpiresAt: r.ExpiresAt}, fields
}

func currentUser(c *gin.Context) auth.User { return c.MustGet("session").(auth.Session).User }
func cleanList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
func validStatus(value string) bool { return oneOf(value, "draft", "published", "paused", "closed") }
func writeJobValidation(c *gin.Context, fields map[string]string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "Vacancy data is invalid", "fields": fields}})
}
func writeEmployerJobError(c *gin.Context, logger *slog.Logger, err error) {
	switch {
	case errors.Is(err, jobs.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "JOB_NOT_FOUND", "message": "Job not found"}})
	case errors.Is(err, jobs.ErrInvalidTransition):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "INVALID_JOB_TRANSITION", "message": "Job status transition is not allowed"}})
	default:
		logger.Error("employer job operation failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "JOBS_UNAVAILABLE", "message": "Jobs are temporarily unavailable"}})
	}
}
