package handlers

import (
	"github.com/gin-gonic/gin"
	"jobhub-ai/backend/internal/auth"
	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/middleware"
	"log/slog"
	"net/http"
)

func NewRouter(health HealthChecker, logger *slog.Logger, origin string, extras ...any) *gin.Engine {
	var jobReader jobs.Reader
	var employerJobs jobs.EmployerStore
	var authService auth.Service
	environment := "development"
	for _, extra := range extras {
		if value, ok := extra.(jobs.Reader); ok {
			jobReader = value
		}
		if value, ok := extra.(jobs.EmployerStore); ok {
			employerJobs = value
		}
		if value, ok := extra.(auth.Service); ok {
			authService = value
		}
		if value, ok := extra.(string); ok {
			environment = value
		}
	}
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(middleware.HTTP(logger, origin))
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error("request panic", "panic", recovered)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "An unexpected error occurred"}})
	}))
	router.GET("/api/v1/health", Health(health, logger))
	if jobReader != nil {
		router.GET("/api/v1/jobs", ListJobs(jobReader, logger))
		router.GET("/api/v1/jobs/:id", GetJob(jobReader, logger))
	}
	if authService != nil {
		RegisterAuthRoutes(router, authService, logger, origin, environment)
		router.GET("/api/v1/employer/me", RequireRole(authService, auth.RoleEmployer), func(c *gin.Context) {
			session := c.MustGet("session").(auth.Session)
			c.JSON(http.StatusOK, gin.H{"user": session.User, "company": session.Company})
		})
		router.GET("/api/v1/admin/me", RequireRole(authService, auth.RoleAdmin), func(c *gin.Context) {
			session := c.MustGet("session").(auth.Session)
			c.JSON(http.StatusOK, gin.H{"user": session.User})
		})
	}
	if authService != nil && employerJobs != nil {
		RegisterEmployerJobRoutes(router, employerJobs, authService, logger, origin)
	}
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Endpoint not found"}})
	})
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": gin.H{"code": "METHOD_NOT_ALLOWED", "message": "Method not allowed"}})
	})
	return router
}
