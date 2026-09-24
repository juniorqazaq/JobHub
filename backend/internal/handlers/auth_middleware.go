package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"jobhub-ai/backend/internal/auth"
)

func RequireRole(service auth.Service, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie(auth.SessionCookieName)
		session, err := service.Authenticate(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHENTICATED", "message": "Authentication required"}})
			return
		}
		if session.User.Role != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Role is not allowed"}})
			return
		}
		c.Set("session", session)
		c.Next()
	}
}
