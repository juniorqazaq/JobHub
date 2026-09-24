package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"jobhub-ai/backend/internal/auth"
)

type authService interface {
	Register(c *gin.Context, input auth.RegisterInput) (auth.Session, string, error)
}

type authHandler struct {
	service auth.Service
	logger  *slog.Logger
	origin  string
	secure  bool
}

type registerRequest struct {
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Role        string `json:"role"`
	CompanyName string `json:"company_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type sessionResponse struct {
	User      auth.User     `json:"user"`
	Company   *auth.Company `json:"company,omitempty"`
	CSRFToken string        `json:"csrf_token"`
	ExpiresAt time.Time     `json:"expires_at"`
}

func RegisterAuthRoutes(router *gin.Engine, service auth.Service, logger *slog.Logger, origin, environment string) {
	h := authHandler{service: service, logger: logger, origin: origin, secure: environment == "production"}
	group := router.Group("/api/v1/auth")
	group.POST("/register", h.requireTrustedOrigin(), h.register)
	group.POST("/login", h.requireTrustedOrigin(), h.login)
	group.GET("/me", h.me)
	group.POST("/logout", h.requireTrustedOrigin(), h.requireCSRF(), h.logout)
}

func (h authHandler) register(c *gin.Context) {
	var req registerRequest
	if !decodeJSON(c, &req) {
		return
	}
	session, token, err := h.service.Register(c.Request.Context(), auth.RegisterInput{
		FullName: req.FullName, Email: req.Email, Password: req.Password, Role: req.Role, CompanyName: req.CompanyName,
	})
	if err != nil {
		h.writeAuthError(c, err)
		return
	}
	h.setSessionCookie(c, token, session.ExpiresAt)
	c.JSON(http.StatusCreated, mapSessionResponse(session))
}

func (h authHandler) login(c *gin.Context) {
	var req loginRequest
	if !decodeJSON(c, &req) {
		return
	}
	session, token, err := h.service.Login(c.Request.Context(), auth.LoginInput{Email: req.Email, Password: req.Password})
	if err != nil {
		h.writeAuthError(c, err)
		return
	}
	h.setSessionCookie(c, token, session.ExpiresAt)
	c.JSON(http.StatusOK, mapSessionResponse(session))
}

func (h authHandler) me(c *gin.Context) {
	token, _ := c.Cookie(auth.SessionCookieName)
	session, err := h.service.Authenticate(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHENTICATED", "message": "Authentication required"}})
		return
	}
	c.JSON(http.StatusOK, mapSessionResponse(session))
}

func (h authHandler) logout(c *gin.Context) {
	token, _ := c.Cookie(auth.SessionCookieName)
	if err := h.service.Logout(c.Request.Context(), token); err != nil {
		h.logger.Error("logout failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "AUTH_UNAVAILABLE", "message": "Authentication is temporarily unavailable"}})
		return
	}
	h.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

func (h authHandler) requireTrustedOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && origin != h.origin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN_ORIGIN", "message": "Request origin is not trusted"}})
			return
		}
		c.Next()
	}
}

func (h authHandler) requireCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie(auth.SessionCookieName)
		if err := h.service.ValidateCSRF(c.Request.Context(), token, c.GetHeader(auth.CSRFHeaderName)); err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "CSRF_INVALID", "message": "CSRF token is invalid"}})
			return
		}
		c.Next()
	}
}

func (h authHandler) writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, auth.ErrDuplicateEmail):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "EMAIL_ALREADY_EXISTS", "message": "Email is already registered"}})
	case errors.Is(err, auth.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "INVALID_CREDENTIALS", "message": "Invalid email or password"}})
	case errors.Is(err, auth.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Action is not allowed"}})
	case errors.Is(err, auth.ErrSuspended):
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "ACCOUNT_SUSPENDED", "message": "Account is suspended"}})
	default:
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "password") {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
			return
		}
		h.logger.Error("auth request failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "AUTH_UNAVAILABLE", "message": "Authentication is temporarily unavailable"}})
	}
}

func (h authHandler) setSessionCookie(c *gin.Context, token string, expiresAt time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: auth.SessionCookieName, Value: token, Path: "/", Expires: expiresAt,
		HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode,
	})
}

func (h authHandler) clearSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: auth.SessionCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode,
	})
}

func mapSessionResponse(session auth.Session) sessionResponse {
	return sessionResponse{User: session.User, Company: session.Company, CSRFToken: session.CSRFToken, ExpiresAt: session.ExpiresAt}
}
