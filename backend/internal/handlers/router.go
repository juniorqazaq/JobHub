package handlers

import (
	"github.com/gin-gonic/gin"
	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/middleware"
	"log/slog"
	"net/http"
)

func NewRouter(health HealthChecker, logger *slog.Logger, origin string, jobReaders ...jobs.Reader) *gin.Engine {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(middleware.HTTP(logger, origin))
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error("request panic", "panic", recovered)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "An unexpected error occurred"}})
	}))
	router.GET("/api/v1/health", Health(health, logger))
	if len(jobReaders) > 0 && jobReaders[0] != nil {
		router.GET("/api/v1/jobs", ListJobs(jobReaders[0], logger))
		router.GET("/api/v1/jobs/:id", GetJob(jobReaders[0], logger))
	}
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Endpoint not found"}})
	})
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": gin.H{"code": "METHOD_NOT_ALLOWED", "message": "Method not allowed"}})
	})
	return router
}
