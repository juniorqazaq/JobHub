-- name: CreateUser :one
INSERT INTO jobhub.users (full_name, email, normalized_email, password_hash, role)
VALUES (sqlc.arg(full_name), sqlc.arg(email), sqlc.arg(normalized_email), sqlc.arg(password_hash), sqlc.arg(role))
RETURNING *;

-- name: GetUserByNormalizedEmail :one
SELECT * FROM jobhub.users
WHERE normalized_email = sqlc.arg(normalized_email);

-- name: GetUserByID :one
SELECT * FROM jobhub.users
WHERE id = sqlc.arg(id);

-- name: CreateCompany :one
INSERT INTO jobhub.companies (name)
VALUES (sqlc.arg(name))
RETURNING *;

-- name: CreateCompanyMembership :one
INSERT INTO jobhub.company_memberships (company_id, user_id, role)
VALUES (sqlc.arg(company_id), sqlc.arg(user_id), sqlc.arg(role))
RETURNING *;

-- name: GetCompanyForUser :one
SELECT c.*
FROM jobhub.companies c
JOIN jobhub.company_memberships cm ON cm.company_id = c.id
WHERE cm.user_id = sqlc.arg(user_id)
ORDER BY cm.created_at ASC
LIMIT 1;

-- name: CreateSession :one
INSERT INTO jobhub.sessions (user_id, token_hash, csrf_hash, expires_at)
VALUES (sqlc.arg(user_id), sqlc.arg(token_hash), sqlc.arg(csrf_hash), sqlc.arg(expires_at))
RETURNING *;

-- name: GetSessionByTokenHash :one
SELECT sqlc.embed(s), sqlc.embed(u)
FROM jobhub.sessions s
JOIN jobhub.users u ON u.id = s.user_id
WHERE s.token_hash = sqlc.arg(token_hash);

-- name: RevokeSession :exec
UPDATE jobhub.sessions
SET revoked_at = COALESCE(revoked_at, now())
WHERE id = sqlc.arg(id);

-- name: DeleteExpiredSessions :exec
DELETE FROM jobhub.sessions
WHERE expires_at < now() OR revoked_at IS NOT NULL;
