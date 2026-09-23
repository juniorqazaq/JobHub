package handlers

import (
	"context"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
)

type HealthChecker interface{ Check(context.Context) error }

func Health(service HealthChecker, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := service.Check(c.Request.Context()); err != nil {
			logger.ErrorContext(c.Request.Context(), "database health check failed", "error", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "DATABASE_UNAVAILABLE", "message": "Database is unavailable"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "database": "up", "service": "jobhub-ai"})
	}
}
