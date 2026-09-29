package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"jobhub-ai/backend/internal/auth"
	"jobhub-ai/backend/internal/companies"
)

type companySearchDTO struct {
	Items      []companies.Company `json:"items"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	Total      int64               `json:"total"`
	TotalPages int                 `json:"total_pages"`
}

type companyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	WebsiteURL  string `json:"website_url"`
	LogoURL     string `json:"logo_url"`
	Industry    string `json:"industry"`
	City        string `json:"city"`
}

func RegisterCompanyRoutes(router *gin.Engine, store companies.Store, authService auth.Service, logger *slog.Logger, origin string) {
	router.GET("/api/v1/companies", listCompanies(store, logger))
	router.GET("/api/v1/companies/:id", getCompany(store, logger))
	if authService == nil {
		return
	}

	employerRead := []gin.HandlerFunc{RequireRole(authService, auth.RoleEmployer)}
	employerWrite := []gin.HandlerFunc{RequireTrustedOrigin(origin), RequireRole(authService, auth.RoleEmployer), RequireCSRF(authService)}
	router.GET("/api/v1/employer/company", append(employerRead, getEmployerCompany(store, logger))...)
	router.PATCH("/api/v1/employer/company", append(employerWrite, updateEmployerCompany(store, logger))...)

	candidateRead := []gin.HandlerFunc{RequireRole(authService, auth.RoleJobSeeker)}
	candidateWrite := []gin.HandlerFunc{RequireTrustedOrigin(origin), RequireRole(authService, auth.RoleJobSeeker), RequireCSRF(authService)}
	router.GET("/api/v1/companies/:id/follow", append(candidateRead, companyFollowState(store, logger))...)
	router.PUT("/api/v1/companies/:id/follow", append(candidateWrite, followCompany(store, logger))...)
	router.DELETE("/api/v1/companies/:id/follow", append(candidateWrite, unfollowCompany(store, logger))...)
}

func listCompanies(store companies.Store, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := positiveInt(c, "page", 1, 1, 100000)
		if !ok {
			return
		}
		pageSize, ok := positiveInt(c, "page_size", 20, 1, 100)
		if !ok {
			return
		}
		query := strings.TrimSpace(c.Query("q"))
		if utf8.RuneCountInString(query) > 120 {
			validationError(c, "q")
			return
		}
		result, err := store.Search(c.Request.Context(), query, page, pageSize)
		if err != nil {
			writeCompanyError(c, logger, err)
			return
		}
		totalPages := 0
		if result.Total > 0 {
			totalPages = int((result.Total + int64(pageSize) - 1) / int64(pageSize))
		}
		c.JSON(http.StatusOK, companySearchDTO{Items: result.Items, Page: page, PageSize: pageSize, Total: result.Total, TotalPages: totalPages})
	}
}

func getCompany(store companies.Store, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		detail, err := store.GetPublic(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeCompanyError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, detail)
	}
}

func getEmployerCompany(store companies.Store, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		company, err := store.GetForEmployer(c.Request.Context(), currentUser(c).ID)
		if err != nil {
			writeCompanyError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, company)
	}
}

func updateEmployerCompany(store companies.Store, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request companyRequest
		if !decodeJSON(c, &request) {
			return
		}
		input, fields := validateCompanyRequest(request)
		if len(fields) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "Company data is invalid", "fields": fields}})
			return
		}
		company, err := store.UpdateForEmployer(c.Request.Context(), currentUser(c).ID, input)
		if err != nil {
			writeCompanyError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, company)
	}
}

func companyFollowState(store companies.Store, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		following, err := store.IsFollowing(c.Request.Context(), currentUser(c).ID, c.Param("id"))
		if err != nil {
			writeCompanyError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"following": following})
	}
}

func followCompany(store companies.Store, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := store.Follow(c.Request.Context(), currentUser(c).ID, c.Param("id")); err != nil {
			writeCompanyError(c, logger, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func unfollowCompany(store companies.Store, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := store.Unfollow(c.Request.Context(), currentUser(c).ID, c.Param("id")); err != nil {
			writeCompanyError(c, logger, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func validateCompanyRequest(request companyRequest) (companies.UpdateInput, map[string]string) {
	fields := map[string]string{}
	request.Name = strings.TrimSpace(request.Name)
	request.Description = strings.TrimSpace(request.Description)
	request.WebsiteURL = strings.TrimSpace(request.WebsiteURL)
	request.LogoURL = strings.TrimSpace(request.LogoURL)
	request.Industry = strings.TrimSpace(request.Industry)
	request.City = strings.TrimSpace(request.City)
	validateText(fields, "name", request.Name, 2, 160, true)
	validateText(fields, "description", request.Description, 0, 5000, false)
	validateText(fields, "industry", request.Industry, 0, 120, false)
	validateText(fields, "city", request.City, 0, 120, false)
	if !validOptionalURL(request.WebsiteURL, false) {
		fields["website_url"] = "must be a valid HTTP(S) URL"
	}
	if !validOptionalURL(request.LogoURL, true) {
		fields["logo_url"] = "must be an HTTPS or local URL"
	}
	return companies.UpdateInput{Name: request.Name, Description: request.Description, WebsiteURL: request.WebsiteURL,
		LogoURL: request.LogoURL, Industry: request.Industry, City: request.City}, fields
}

func validOptionalURL(value string, allowLocal bool) bool {
	if value == "" {
		return true
	}
	if utf8.RuneCountInString(value) > 500 {
		return false
	}
	if allowLocal && strings.HasPrefix(value, "/company-logos/") && !strings.Contains(value, "..") {
		return true
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return false
	}
	if allowLocal {
		return parsed.Scheme == "https"
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func writeCompanyError(c *gin.Context, logger *slog.Logger, err error) {
	if errors.Is(err, companies.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "COMPANY_NOT_FOUND", "message": "Company not found"}})
		return
	}
	logger.Error("company operation failed", "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "COMPANIES_UNAVAILABLE", "message": "Companies are temporarily unavailable"}})
}
