package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"jobhub-ai/backend/internal/database/dbgen"
)

type AuthService struct {
	pool       *pgxpool.Pool
	queries    *dbgen.Queries
	sessionTTL time.Duration
	now        func() time.Time
}

func NewService(pool *pgxpool.Pool) *AuthService {
	return &AuthService{pool: pool, queries: dbgen.New(pool), sessionTTL: 7 * 24 * time.Hour, now: time.Now}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (Session, string, error) {
	if err := validateRegistration(input); err != nil {
		return Session{}, "", err
	}
	normalizedEmail, err := NormalizeEmail(input.Email)
	if err != nil {
		return Session{}, "", err
	}
	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return Session{}, "", err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Session{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := dbgen.New(tx)
	user, err := q.CreateUser(ctx, dbgen.CreateUserParams{
		FullName: strings.TrimSpace(input.FullName), Email: strings.TrimSpace(input.Email),
		NormalizedEmail: normalizedEmail, PasswordHash: passwordHash, Role: input.Role,
	})
	if isUniqueViolation(err) {
		return Session{}, "", ErrDuplicateEmail
	}
	if err != nil {
		return Session{}, "", fmt.Errorf("create user: %w", err)
	}
	var company *Company
	if input.Role == RoleEmployer {
		created, err := q.CreateCompany(ctx, strings.TrimSpace(input.CompanyName))
		if err != nil {
			return Session{}, "", fmt.Errorf("create company: %w", err)
		}
		if _, err := q.CreateCompanyMembership(ctx, dbgen.CreateCompanyMembershipParams{CompanyID: created.ID, UserID: user.ID, Role: "owner"}); err != nil {
			return Session{}, "", fmt.Errorf("create company membership: %w", err)
		}
		company = &Company{ID: formatUUID(created.ID), Name: created.Name}
	}
	session, token, err := s.createSession(ctx, q, user, company)
	if err != nil {
		return Session{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, "", fmt.Errorf("commit registration: %w", err)
	}
	return session, token, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (Session, string, error) {
	normalizedEmail, err := NormalizeEmail(input.Email)
	if err != nil {
		return Session{}, "", ErrInvalidCredentials
	}
	user, err := s.queries.GetUserByNormalizedEmail(ctx, normalizedEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, "", fmt.Errorf("get user: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		return Session{}, "", ErrInvalidCredentials
	}
	if user.Status != StatusActive {
		return Session{}, "", ErrSuspended
	}
	var company *Company
	if user.Role == RoleEmployer {
		if c, err := s.queries.GetCompanyForUser(ctx, user.ID); err == nil {
			company = &Company{ID: formatUUID(c.ID), Name: c.Name}
		}
	}
	session, token, err := s.createSession(ctx, s.queries, user, company)
	if err != nil {
		return Session{}, "", err
	}
	return session, token, nil
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (Session, error) {
	if token == "" {
		return Session{}, ErrInvalidSession
	}
	row, err := s.queries.GetSessionByTokenHash(ctx, tokenHash(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrInvalidSession
	}
	if err != nil {
		return Session{}, fmt.Errorf("get session: %w", err)
	}
	if row.JobhubSession.RevokedAt.Valid || !row.JobhubSession.ExpiresAt.Time.After(s.now().UTC()) {
		return Session{}, ErrInvalidSession
	}
	if row.JobhubUser.Status != StatusActive {
		return Session{}, ErrSuspended
	}
	session := mapSession(row.JobhubSession, row.JobhubUser, "")
	if row.JobhubUser.Role == RoleEmployer {
		if c, err := s.queries.GetCompanyForUser(ctx, row.JobhubUser.ID); err == nil {
			session.Company = &Company{ID: formatUUID(c.ID), Name: c.Name}
		}
	}
	return session, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	row, err := s.queries.GetSessionByTokenHash(ctx, tokenHash(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	return s.queries.RevokeSession(ctx, row.JobhubSession.ID)
}

func (s *AuthService) ValidateCSRF(ctx context.Context, token, submitted string) error {
	if token == "" || submitted == "" {
		return ErrForbidden
	}
	row, err := s.queries.GetSessionByTokenHash(ctx, tokenHash(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidSession
	}
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	if row.JobhubSession.RevokedAt.Valid || !row.JobhubSession.ExpiresAt.Time.After(s.now().UTC()) {
		return ErrInvalidSession
	}
	if row.JobhubUser.Status != StatusActive {
		return ErrSuspended
	}
	session := mapSession(row.JobhubSession, row.JobhubUser, "")
	if !ValidateCSRF(session, submitted, row.JobhubSession.CsrfHash) {
		return ErrForbidden
	}
	return nil
}

func (s *AuthService) CreateAdmin(ctx context.Context, input AdminInput) (User, error) {
	if strings.TrimSpace(input.FullName) == "" || len(input.Password) < 12 {
		return User{}, fmt.Errorf("admin full name and password of at least 12 characters are required")
	}
	normalizedEmail, err := NormalizeEmail(input.Email)
	if err != nil {
		return User{}, err
	}
	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return User{}, err
	}
	user, err := s.queries.CreateUser(ctx, dbgen.CreateUserParams{
		FullName: strings.TrimSpace(input.FullName), Email: strings.TrimSpace(input.Email),
		NormalizedEmail: normalizedEmail, PasswordHash: passwordHash, Role: RoleAdmin,
	})
	if isUniqueViolation(err) {
		return User{}, ErrDuplicateEmail
	}
	if err != nil {
		return User{}, err
	}
	return mapUser(user), nil
}

func (s *AuthService) createSession(ctx context.Context, q *dbgen.Queries, user dbgen.JobhubUser, company *Company) (Session, string, error) {
	token, err := randomToken()
	if err != nil {
		return Session{}, "", err
	}
	csrf, err := randomToken()
	if err != nil {
		return Session{}, "", err
	}
	expiresAt := s.now().UTC().Add(s.sessionTTL)
	row, err := q.CreateSession(ctx, dbgen.CreateSessionParams{
		UserID: user.ID, TokenHash: tokenHash(token), CsrfHash: tokenHash(csrf),
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return Session{}, "", fmt.Errorf("create session: %w", err)
	}
	session := mapSession(row, user, csrf)
	session.Company = company
	return session, token, nil
}

func ValidateCSRF(session Session, submitted string, storedHash []byte) bool {
	if submitted == "" || len(storedHash) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare(tokenHash(submitted), storedHash) == 1 && session.ID != ""
}

func NormalizeEmail(email string) (string, error) {
	parsed, err := mail.ParseAddress(strings.TrimSpace(email))
	if err != nil || strings.Contains(parsed.Address, " ") {
		return "", fmt.Errorf("invalid email")
	}
	return strings.ToLower(parsed.Address), nil
}

func validateRegistration(input RegisterInput) error {
	if strings.TrimSpace(input.FullName) == "" {
		return fmt.Errorf("full_name is required")
	}
	if _, err := NormalizeEmail(input.Email); err != nil {
		return fmt.Errorf("email is invalid")
	}
	if len(input.Password) < 10 {
		return fmt.Errorf("password must be at least 10 characters")
	}
	if input.Role != RoleJobSeeker && input.Role != RoleEmployer {
		return ErrForbidden
	}
	if input.Role == RoleEmployer && strings.TrimSpace(input.CompanyName) == "" {
		return fmt.Errorf("company_name is required")
	}
	return nil
}

func hashPassword(password string) (string, error) {
	value, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(value), err
}

func tokenHash(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func mapSession(session dbgen.JobhubSession, user dbgen.JobhubUser, csrf string) Session {
	return Session{ID: formatUUID(session.ID), User: mapUser(user), CSRFToken: csrf, ExpiresAt: session.ExpiresAt.Time}
}

func mapUser(user dbgen.JobhubUser) User {
	return User{ID: formatUUID(user.ID), FullName: user.FullName, Email: user.Email, Role: user.Role, Status: user.Status}
}

func formatUUID(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	b := value.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func parseUUID(value string) (pgtype.UUID, error) {
	decoded, err := hex.DecodeString(strings.ReplaceAll(value, "-", ""))
	if err != nil || len(decoded) != 16 {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID")
	}
	var bytes [16]byte
	copy(bytes[:], decoded)
	return pgtype.UUID{Bytes: bytes, Valid: true}, nil
}
