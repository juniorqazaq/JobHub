package handlers

import (
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"jobhub-ai/backend/internal/auth"
	"jobhub-ai/backend/internal/candidates"
	"jobhub-ai/backend/internal/locations"
)

type profileRequest struct {
	FullName                 string                      `json:"full_name"`
	PhotoURL                 string                      `json:"photo_url"`
	City                     string                      `json:"city"`
	CityID                   string                      `json:"city_id"`
	BirthYear                *int                        `json:"birth_year"`
	BirthDate                string                      `json:"birth_date"`
	Phone                    string                      `json:"phone"`
	About                    string                      `json:"about"`
	CurrentPosition          string                      `json:"current_position"`
	DesiredPosition          string                      `json:"desired_position"`
	YearsExperience          *float64                    `json:"years_experience"`
	ExperienceLevel          string                      `json:"experience_level"`
	Skills                   []string                    `json:"skills"`
	Certifications           []string                    `json:"certifications"`
	Languages                []candidates.Language       `json:"languages"`
	DesiredSalary            *float64                    `json:"desired_salary"`
	Currency                 string                      `json:"currency"`
	SalaryPeriod             string                      `json:"salary_period"`
	PreferredLocations       []string                    `json:"preferred_locations"`
	PreferredCityIDs         []string                    `json:"preferred_city_ids"`
	PreferredEmploymentTypes []string                    `json:"preferred_employment_types"`
	PreferredWorkModes       []string                    `json:"preferred_work_modes"`
	PreferredCategories      []string                    `json:"preferred_categories"`
	PreferredRoles           []string                    `json:"preferred_roles"`
	SearchStatus             string                      `json:"search_status"`
	GitHub                   string                      `json:"github"`
	LinkedIn                 string                      `json:"linkedin"`
	Portfolio                string                      `json:"portfolio"`
	Website                  string                      `json:"website"`
	AllowEmployerContact     bool                        `json:"allow_employer_contact"`
	ShowProfileToEmployers   bool                        `json:"show_profile_to_employers"`
	ShowSalaryExpectations   bool                        `json:"show_salary_expectations"`
	WorkExperience           []candidates.WorkExperience `json:"work_experience"`
	Education                []candidates.Education      `json:"education"`
}

func RegisterCandidateRoutes(router *gin.Engine, service *candidates.Service, authService auth.Service, logger *slog.Logger, origin string) {
	read := []gin.HandlerFunc{RequireRole(authService, auth.RoleJobSeeker)}
	write := []gin.HandlerFunc{RequireTrustedOrigin(origin), RequireRole(authService, auth.RoleJobSeeker), RequireCSRF(authService)}
	employerRead := []gin.HandlerFunc{RequireRole(authService, auth.RoleEmployer)}
	employerWrite := []gin.HandlerFunc{RequireTrustedOrigin(origin), RequireRole(authService, auth.RoleEmployer), RequireCSRF(authService)}

	router.GET("/api/v1/profile", append(read, getProfile(service, logger))...)
	router.PATCH("/api/v1/profile", append(write, updateProfile(service, logger))...)
	router.POST("/api/v1/profile/resume", append(write, uploadResume(service, logger))...)
	router.GET("/api/v1/profile/resume", append(read, getResume(service, logger))...)
	router.GET("/api/v1/profile/resume/content", append(read, ownResumeContent(service, logger))...)
	router.DELETE("/api/v1/profile/resume", append(write, deleteResume(service, logger))...)
	router.GET("/api/v1/saved-jobs", append(read, listSavedJobs(service, logger))...)
	router.PUT("/api/v1/jobs/:id/saved", append(write, saveJob(service, logger))...)
	router.DELETE("/api/v1/jobs/:id/saved", append(write, unsaveJob(service, logger))...)
	router.POST("/api/v1/jobs/:id/applications", append(write, createApplication(service, logger))...)
	router.GET("/api/v1/applications", append(read, listApplications(service, logger))...)
	router.PATCH("/api/v1/applications/:id/withdraw", append(write, withdrawApplication(service, logger))...)
	router.GET("/api/v1/employer/jobs/:id/applications", append(employerRead, listEmployerApplications(service, logger))...)
	router.GET("/api/v1/employer/applications/:id/resume", append(employerRead, employerResumeContent(service, logger))...)
	router.PATCH("/api/v1/employer/applications/:id", append(employerWrite, updateEmployerApplication(service, logger))...)
}

func getProfile(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := service.Store().GetProfile(c.Request.Context(), currentUser(c).ID)
		if err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

func updateProfile(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request profileRequest
		if !decodeJSON(c, &request) {
			return
		}
		p, fields := validateProfile(request)
		if len(fields) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "Profile data is invalid", "fields": fields}})
			return
		}
		updated, err := service.Store().UpdateProfile(c.Request.Context(), currentUser(c).ID, p)
		if err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, updated)
	}
}

func uploadResume(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, candidates.MaxResumeBytes+(1<<20))
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeCandidateError(c, logger, candidates.ErrResumeTooLarge)
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "RESUME_REQUIRED", "message": "PDF resume is required"}})
			}
			return
		}
		defer file.Close()
		resume, err := service.UploadResume(c.Request.Context(), currentUser(c).ID, header.Filename, header.Header.Get("Content-Type"), file)
		if err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		c.JSON(http.StatusCreated, resume)
	}
}

func getResume(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, err := service.Store().GetActiveResume(c.Request.Context(), currentUser(c).ID)
		if err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, r)
	}
}

func ownResumeContent(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, file, err := service.OpenOwnResume(c.Request.Context(), currentUser(c).ID)
		if err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		defer file.Close()
		sendResume(c, r, file, "inline")
	}
}

func deleteResume(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := service.DeleteOwnResume(c.Request.Context(), currentUser(c).ID); err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func listSavedJobs(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := service.Store().ListSavedJobs(c.Request.Context(), currentUser(c).ID)
		if err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func saveJob(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := service.Store().SaveJob(c.Request.Context(), currentUser(c).ID, c.Param("id")); err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}
func unsaveJob(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := service.Store().UnsaveJob(c.Request.Context(), currentUser(c).ID, c.Param("id")); err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

type applicationRequest struct {
	ResumeID string `json:"resume_id"`
	Message  string `json:"message"`
}

func createApplication(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request applicationRequest
		if !decodeJSON(c, &request) {
			return
		}
		request.ResumeID = strings.TrimSpace(request.ResumeID)
		request.Message = strings.TrimSpace(request.Message)
		if request.ResumeID == "" || len(request.Message) > 3000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "Application data is invalid"}})
			return
		}
		item, err := service.Store().CreateApplication(c.Request.Context(), currentUser(c).ID, c.Param("id"), request.ResumeID, request.Message)
		if errors.Is(err, candidates.ErrConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "APPLICATION_EXISTS", "message": "Application already exists", "application_id": item.ID}})
			return
		}
		if err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		c.JSON(http.StatusCreated, item)
	}
}

func listApplications(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := service.Store().ListCandidateApplications(c.Request.Context(), currentUser(c).ID)
		if err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}
func withdrawApplication(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := service.Store().WithdrawApplication(c.Request.Context(), currentUser(c).ID, c.Param("id"))
		if err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, item)
	}
}
func listEmployerApplications(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := service.Store().ListEmployerApplications(c.Request.Context(), currentUser(c).ID, c.Param("id"))
		if err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

type statusRequest struct {
	Status string `json:"status"`
}

func updateEmployerApplication(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request statusRequest
		if !decodeJSON(c, &request) {
			return
		}
		if !oneOf(request.Status, "viewed", "in_review", "contacted", "interview", "offer", "rejected") {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "Application status is invalid"}})
			return
		}
		item, err := service.Store().UpdateEmployerApplication(c.Request.Context(), currentUser(c).ID, c.Param("id"), request.Status)
		if err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, item)
	}
}
func employerResumeContent(service *candidates.Service, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, file, err := service.OpenEmployerResume(c.Request.Context(), currentUser(c).ID, c.Param("id"))
		if err != nil {
			writeCandidateError(c, logger, err)
			return
		}
		defer file.Close()
		sendResume(c, r, file, "attachment")
	}
}

func sendResume(c *gin.Context, r candidates.Resume, file io.Reader, disposition string) {
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": r.OriginalFilename}))
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, file)
}

func validateProfile(r profileRequest) (candidates.Profile, map[string]string) {
	fields := map[string]string{}
	nowYear := time.Now().Year()
	r.FullName = strings.TrimSpace(r.FullName)
	if r.FullName == "" || len(r.FullName) > 160 {
		fields["full_name"] = "required and at most 160 characters"
	}
	if r.BirthYear != nil && (*r.BirthYear < 1900 || *r.BirthYear > nowYear) {
		fields["birth_year"] = "invalid year"
	}
	r.BirthDate = strings.TrimSpace(r.BirthDate)
	if r.BirthDate != "" {
		birthDate, err := time.Parse("2006-01-02", r.BirthDate)
		if err != nil || birthDate.After(time.Now()) || birthDate.Year() < 1900 {
			fields["birth_date"] = "must be a valid date that is not in the future"
		}
	}
	phone, phoneErr := normalizeProfilePhone(r.Phone)
	if phoneErr != nil {
		fields["phone"] = "must be a valid international or Kazakhstan phone number"
	} else {
		r.Phone = phone
	}
	if r.YearsExperience != nil && (*r.YearsExperience < 0 || *r.YearsExperience > 80) {
		fields["years_experience"] = "must be between 0 and 80"
	}
	if r.ExperienceLevel != "" && !oneOf(r.ExperienceLevel, "internship", "junior", "middle", "senior", "lead") {
		fields["experience_level"] = "invalid value"
	}
	if r.CityID != "" && !locations.Valid(r.CityID) {
		fields["city_id"] = "invalid value"
	}
	seenCities := map[string]bool{}
	for _, cityID := range r.PreferredCityIDs {
		if !locations.Valid(cityID) || seenCities[cityID] {
			fields["preferred_city_ids"] = "contains invalid or duplicate values"
		}
		seenCities[cityID] = true
	}
	if !oneOf(r.SearchStatus, "actively_looking", "open_to_offers", "not_looking") {
		fields["search_status"] = "invalid value"
	}
	if r.DesiredSalary != nil && *r.DesiredSalary < 0 {
		fields["desired_salary"] = "must be non-negative"
	}
	if r.SalaryPeriod != "" && !oneOf(r.SalaryPeriod, "month", "year") {
		fields["salary_period"] = "invalid value"
	}
	for name, value := range map[string]string{"photo_url": r.PhotoURL, "github": r.GitHub, "linkedin": r.LinkedIn, "portfolio": r.Portfolio, "website": r.Website} {
		if value != "" && !httpURL(value) {
			fields[name] = "must be an http(s) URL"
		}
	}
	seenLanguages := map[string]bool{}
	for i := range r.Languages {
		item := &r.Languages[i]
		item.Language = strings.TrimSpace(item.Language)
		key := strings.ToLower(item.Language)
		if item.Language == "" || seenLanguages[key] || !oneOf(item.Proficiency, "native", "fluent", "A1", "A2", "B1", "B2", "C1", "C2") {
			fields["languages"] = "contains invalid or duplicate entries"
		}
		seenLanguages[key] = true
	}
	for i := range r.WorkExperience {
		item := &r.WorkExperience[i]
		start, startErr := time.Parse("2006-01-02", item.StartDate)
		if strings.TrimSpace(item.Company) == "" || strings.TrimSpace(item.Position) == "" || startErr != nil || !oneOf(item.EmploymentType, "full_time", "part_time", "contract", "temporary", "internship") {
			fields["work_experience"] = "contains invalid entries"
			continue
		}
		if item.IsCurrent {
			item.EndDate = ""
		} else {
			end, endErr := time.Parse("2006-01-02", item.EndDate)
			if endErr != nil || end.Before(start) {
				fields["work_experience"] = "end date must follow start date"
			}
		}
		item.Skills = cleanList(item.Skills)
	}
	for _, item := range r.Education {
		if strings.TrimSpace(item.Institution) == "" || strings.TrimSpace(item.Degree) == "" || strings.TrimSpace(item.FieldOfStudy) == "" || item.StartYear < 1900 || item.StartYear > 2100 || (item.GraduationYear != nil && (*item.GraduationYear < item.StartYear || *item.GraduationYear > 2100)) {
			fields["education"] = "contains invalid entries"
		}
	}
	city := strings.TrimSpace(r.City)
	if r.CityID != "" {
		city = locations.Name(r.CityID, "en")
	}
	preferredLocations := cleanList(r.PreferredLocations)
	if len(r.PreferredCityIDs) > 0 {
		legacyLocations := make([]string, 0, len(preferredLocations))
		for _, value := range preferredLocations {
			if locations.Normalize(value) == "" {
				legacyLocations = append(legacyLocations, value)
			}
		}
		preferredLocations = legacyLocations
		for _, id := range r.PreferredCityIDs {
			preferredLocations = append(preferredLocations, locations.Name(id, "en"))
		}
	}
	p := candidates.Profile{FullName: r.FullName, PhotoURL: strings.TrimSpace(r.PhotoURL), City: city, CityID: r.CityID, BirthYear: r.BirthYear, BirthDate: r.BirthDate, Phone: r.Phone, About: strings.TrimSpace(r.About), CurrentPosition: strings.TrimSpace(r.CurrentPosition), DesiredPosition: strings.TrimSpace(r.DesiredPosition), YearsExperience: r.YearsExperience, ExperienceLevel: r.ExperienceLevel, Skills: cleanList(r.Skills), Certifications: cleanList(r.Certifications), Languages: r.Languages, DesiredSalary: r.DesiredSalary, Currency: strings.ToUpper(strings.TrimSpace(r.Currency)), SalaryPeriod: r.SalaryPeriod, PreferredLocations: preferredLocations, PreferredCityIDs: r.PreferredCityIDs, PreferredEmploymentTypes: cleanList(r.PreferredEmploymentTypes), PreferredWorkModes: cleanList(r.PreferredWorkModes), PreferredCategories: cleanList(r.PreferredCategories), PreferredRoles: cleanList(r.PreferredRoles), SearchStatus: r.SearchStatus, GitHub: strings.TrimSpace(r.GitHub), LinkedIn: strings.TrimSpace(r.LinkedIn), Portfolio: strings.TrimSpace(r.Portfolio), Website: strings.TrimSpace(r.Website), AllowEmployerContact: r.AllowEmployerContact, ShowProfileToEmployers: r.ShowProfileToEmployers, ShowSalaryExpectations: r.ShowSalaryExpectations, WorkExperience: r.WorkExperience, Education: r.Education}
	return p, fields
}
func httpURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Host != "" && (u.Scheme == "http" || u.Scheme == "https") && u.User == nil
}

func normalizeProfilePhone(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	for _, char := range value {
		if (char < '0' || char > '9') && !strings.ContainsRune("+ ()-", char) {
			return "", errors.New("phone contains unsupported characters")
		}
	}
	digits := strings.NewReplacer(" ", "", "(", "", ")", "", "-", "").Replace(value)
	if strings.HasPrefix(digits, "8") && len(digits) == 11 && (digits[1] == '6' || digits[1] == '7') {
		digits = "+7" + digits[1:]
	} else if !strings.HasPrefix(digits, "+") && len(digits) == 10 && (digits[0] == '6' || digits[0] == '7') {
		digits = "+7" + digits
	}
	if !strings.HasPrefix(digits, "+") || len(digits) < 9 || len(digits) > 16 || digits[1] == '0' {
		return "", errors.New("phone is not E.164 compatible")
	}
	for _, char := range digits[1:] {
		if char < '0' || char > '9' {
			return "", errors.New("phone is not E.164 compatible")
		}
	}
	if strings.HasPrefix(digits, "+7") && (len(digits) != 12 || (digits[2] != '6' && digits[2] != '7')) {
		return "", errors.New("invalid Kazakhstan phone")
	}
	return digits, nil
}

func writeCandidateError(c *gin.Context, logger *slog.Logger, err error) {
	switch {
	case errors.Is(err, candidates.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Resource not found"}})
	case errors.Is(err, candidates.ErrResumeTooLarge):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": gin.H{"code": "RESUME_TOO_LARGE", "message": "Resume exceeds 10 MiB"}})
	case errors.Is(err, candidates.ErrResumeType):
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": gin.H{"code": "RESUME_TYPE_INVALID", "message": "Resume must be a PDF"}})
	case errors.Is(err, candidates.ErrJobUnavailable):
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "JOB_UNAVAILABLE", "message": "Job is not available"}})
	case errors.Is(err, candidates.ErrImportedJob):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "EXTERNAL_JOB", "message": "External jobs use the provider application flow"}})
	case errors.Is(err, candidates.ErrResumeUnavailable):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "RESUME_UNAVAILABLE", "message": "Selected resume is not available"}})
	case errors.Is(err, candidates.ErrInvalidState):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "INVALID_APPLICATION_TRANSITION", "message": "Application status transition is not allowed"}})
	default:
		logger.Error("candidate marketplace operation failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "MARKETPLACE_UNAVAILABLE", "message": "Marketplace operation is temporarily unavailable"}})
	}
}
