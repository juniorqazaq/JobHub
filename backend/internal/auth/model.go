package auth

import (
	"context"
	"errors"
	"time"
)

const (
	RoleJobSeeker = "job_seeker"
	RoleEmployer  = "employer"
	RoleAdmin     = "admin"

	StatusActive    = "active"
	StatusSuspended = "suspended"

	SessionCookieName = "jobhub_session"
	CSRFHeaderName    = "X-CSRF-Token"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrDuplicateEmail     = errors.New("duplicate email")
	ErrInvalidSession     = errors.New("invalid session")
	ErrSuspended          = errors.New("user suspended")
	ErrForbidden          = errors.New("forbidden")
)

type User struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type Company struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Session struct {
	ID        string
	User      User
	Company   *Company
	CSRFToken string
	ExpiresAt time.Time
}

type RegisterInput struct {
	FullName    string
	Email       string
	Password    string
	Role        string
	CompanyName string
}

type LoginInput struct {
	Email    string
	Password string
}

type AdminInput struct {
	FullName string
	Email    string
	Password string
}

type Service interface {
	Register(context.Context, RegisterInput) (Session, string, error)
	Login(context.Context, LoginInput) (Session, string, error)
	Authenticate(context.Context, string) (Session, error)
	ValidateCSRF(context.Context, string, string) error
	Logout(context.Context, string) error
	CreateAdmin(context.Context, AdminInput) (User, error)
}
