package middleware

import (
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"time"
)

func HTTP(logger *slog.Logger, origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Cache-Control", "no-store")
		c.Header("Vary", "Origin")
		if c.GetHeader("Origin") == origin {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
		} else {
			c.Next()
		}
		logger.InfoContext(c.Request.Context(), "http request", "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "duration", time.Since(start))
	}
}
