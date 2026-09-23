# MVP 1: proposed marketplace contract

Status: proposal for approval; none of these endpoints exists yet except health.
This is a Go API extension, not a Supabase Auth or browser database integration.

## Common conventions

- Base path `/api/v1`. UUID identifiers and UTC RFC 3339 timestamps.
- JSON uses snake_case; explicit frontend mappers expose camelCase domain models.
- Single-resource responses return the resource directly. Paginated responses:
  `{ "items": [], "page": 1, "page_size": 20, "total": 0, "total_pages": 0 }`.
- Page and page_size must be positive integers; page_size is capped at 100.
- Keep the existing error envelope and extend it with optional field errors:
  `{ "error": { "code": "VALIDATION_ERROR", "message": "Invalid request", "fields": { "title": "required" } } }`.
  UI maps stable codes to translations; it does not display raw backend errors.
- 400 malformed/invalid payload; 401 missing/expired session; 403 prohibited role;
  404 absent or inaccessible private resource; 409 duplicate/conflicting state;
  413 oversized upload; 415 unsupported file type; 429 rate limit.
- Reject unknown mutation fields, especially role/owner/candidate/verification
  and moderation fields on endpoints where clients cannot set them.
- Mutations return the persisted resource, except idempotent delete/unsave/logout
  operations (204). Creation returns 201 and a Location header.
- Public reads may optionally use a session for saved state. Private routes always
  enforce session role, account status, and ownership in Go and SQL predicates.

## Authentication and role policy

Use Go-managed opaque sessions stored as token hashes in PostgreSQL. Send the
session through an HttpOnly cookie, Secure in production, SameSite=Lax; do not
persist bearer tokens in localStorage. Deploy browser and API on the same site,
preferably with `/api` reverse-proxied. Direct cross-origin development uses an
explicit allowed origin plus credentials, never wildcard CORS.

Require CSRF protection for state-changing requests: enforce trusted Origin and
validate a session-bound token sent via `X-CSRF-Token`. Protect login/register
against cross-site requests too. Logout revokes the database session; rotate
sessions on login. Use an audited password hashing library and rate-limit auth.
Suspended users cannot use existing sessions to perform account actions.

| Method / path | Request | Response / authorization |
| --- | --- | --- |
| POST `/auth/register` | `full_name`, `email`, `password`, `role: job_seeker or employer`, `company_name` required for employer | 201 session response; employer company + owner membership created atomically |
| POST `/auth/login` | `email`, `password` | 200 session response and cookie; generic invalid-credentials error |
| GET `/auth/me` | None | 200 session response, or 401 |
| POST `/auth/logout` | CSRF token | 204 and cookie expiration; revoke session |

Session response:

```json
{
  "user": {
    "id": "<uuid>",
    "full_name": "<name>",
    "email": "<email>",
    "role": "employer",
    "status": "active"
  },
  "csrf_token": "<session-bound-token>",
  "expires_at": "<timestamp>"
}
```

No public request may create or promote an admin, join an arbitrary company,
or set company verification. Provision the initial admin through a backend CLI
run by an authorized operator, with no default password or checked-in seed.
MVP company membership is one owner per newly registered employer; invitations
and association with an existing company are deferred pending verification rules.

For a guest account action, retain an allowlisted internal return URL and action
intent. After auth, return to the same vacancy and application form; do not
automatically submit an application or accept external redirect URLs.

## Companies

| Method / path | Request / query | Response / authorization |
| --- | --- | --- |
| GET `/companies` | `q`, `industry`, `location`, `size`, `sort=relevant or open_jobs`, pagination | Public company summaries; active companies only |
| GET `/companies/:id` | None | Public Company; inactive/private company returns 404 |
| GET `/companies/:id/jobs` | Common job filters | Public job page with the same visibility predicate as `/jobs` |
| GET `/employer/company` | None | Company managed by current employer |
| POST `/companies` | `name`, optional company content below | Employer without a company; creates owner membership; 409 if already associated |
| PATCH `/companies/:id` | Allowed company content fields | Owner only; immutable owner/verification fields |

Company fields: `id`, `name`, `slug`, `description`, `industry`, `headquarters`,
`locations: string[]`, `company_type`, `size`, optional `founded_year`, `website`,
`logo_url`, `benefits: string[]`, server-controlled `is_verified`, `open_job_count`,
`created_at`, `updated_at`. Use validated HTTPS links; never fetch arbitrary
user-supplied URLs on the server. Initial logo/photo management may use validated
URLs; file-upload workflow here is limited to resumes for MVP 1.

Public summaries omit private owner contacts. Verification is false by default.
Open vacancy counts use the public visibility predicate. Ratings, review counts,
recommendation percentages and rank-by-rating are absent until MVP 2 exists.

## Vacancies and search

| Method / path | Behavior / authorization |
| --- | --- |
| GET `/jobs` | Public search; no login required |
| GET `/jobs/:id` | Public visible vacancy detail; hidden/nonpublic resource returns 404 |
| GET `/employer/jobs` | Current employer's vacancies, including drafts and paused jobs |
| GET `/employer/jobs/:id` | Owner-only detail for editing, including moderation status |
| POST `/jobs` | Employer with membership in company_id; draft initially |
| PATCH `/jobs/:id` | Owner-only allowed field updates, including publication lifecycle |
| DELETE `/jobs/:id` | Owner-only soft delete; preserves application and audit records |

Search parameters (same names in shared browser URLs and HTTP requests): `q`,
`location`, repeated `workMode`, repeated `employmentType`, repeated `experience`,
`salaryMin`, `currency`, `postedWithin=1d|3d|7d|30d`, `sort=newest|salary_desc|recommended`,
`page`, `pageSize`, optional `companyId`. Go maps pageSize to response page_size.
Omit empty/default values. Validate all enums and numeric bounds in both layers.

`salaryMin` requires an explicit currency; no comparisons across currencies.
Salary sorting also specifies a currency and groups only comparable periods;
initial search salary filters use monthly salaries. Hidden salaries do not match
salary filters. Sort with deterministic ID tie-breaking. Recommended initially
means text relevance when q is set, otherwise newest; no AI score or personalized
claim. Index/search behavior must be checked against representative data.

Create/update content shape:

```json
{
  "company_id": "<authorized-company-uuid>",
  "title": "Backend Engineer",
  "category": "software_development",
  "description": "<plain text>",
  "responsibilities": ["<responsibility>"],
  "requirements": ["<requirement>"],
  "nice_to_have": [],
  "skills": ["Go", "PostgreSQL"],
  "location": "Almaty",
  "work_mode": "hybrid",
  "employment_type": "full_time",
  "experience_level": "middle",
  "salary": {"min": 600000, "max": 900000, "currency": "KZT", "period": "month"},
  "salary_visible": true,
  "benefits": [],
  "application_method": "internal",
  "expires_at": null
}
```

Employer ownership comes from session + membership, never a supplied user ID.
company_id is fixed after creation. Bounds: nonempty title up to 160 characters,
nonempty description on publishing, bounded text/array lengths, salary min >= 0,
max >= min, allowed currency/enum values, expiration in the future. Drafts allow
incomplete descriptions; publishing requires all necessary fields. Treat content
as plain text, not executable HTML. New jobs are not implicitly verified.

Response adds `id`, `company` summary, `publication_status`, owner-visible
`moderation_status`, `created_at`, `updated_at`, `published_at`, `is_saved`.
List responses include summary metadata; detail includes full content above.
Public responses omit hidden salary amounts and internal moderation details.

Publication: `draft → published → paused/closed`; paused can be republished;
closed is terminal in MVP 1. Soft-deleted jobs cannot be republished.
Moderation is separate: `active`, `pending_review`, `hidden`, `rejected`, `removed`.
New jobs start active moderation; an admin can put them under review or hide them.

A public job must satisfy **all**: published, active moderation, not expired,
not deleted, active company, active employer account. Apply this predicate to
search, details, company vacancies, counts, saved-job availability and application
submission. Save records may remain, but hidden content is returned only as an
unavailable reference to its owner, not as public vacancy content. Employer edits
cannot clear moderation restrictions or set verification.

## Candidate profile and private resume

| Method / path | Behavior / authorization |
| --- | --- |
| GET `/profile` | Current candidate's private profile |
| PATCH `/profile` | Candidate edits own allowed fields |
| POST `/profile/resume` | Candidate uploads/replaces active resume, multipart field `file` |
| GET `/profile/resume` | Active resume metadata, or 404 |
| GET `/profile/resume/content` | Authorized own-file download/preview |
| DELETE `/profile/resume` | Removes active resume from profile; 204 |

Profile: `user_id`, `full_name`, optional `photo_url`, `city`, optional
`birth_year`, `phone`, `about`, `current_position`, `desired_position`,
`years_experience`, `experience_level`, reusable `skills: [{id,name}]`,
`certifications`, `languages: [{language_code,proficiency}]`, links
`{github,linkedin,portfolio,website}`, `search_status` and preferences.
Proficiency: `native`, `fluent`, `A1`, `A2`, `B1`, `B2`, `C1`, `C2`.
Search status: `actively_looking`, `open_to_offers`, `not_looking`.

`work_experience` entries: `id`, `company`, `position`, `employment_type`,
`start_date`, optional `end_date`, `is_current`, `description`, `achievements`,
`skill_ids`. `education` entries: `id`, `institution`, `degree`, `field_of_study`,
`start_year`, optional `graduation_year`, `description`.

`preferences`: salary with currency/period, locations, employment types,
work modes, industries, roles. `privacy`: allow_employer_contact,
show_profile_to_employers, show_salary_expectations, notification_preferences.
Default private; dates of birth and contact details are never in public responses.
MVP 1 has no public candidate-directory endpoint. Privacy defaults do not grant
employers access beyond explicit applications. Return a server-derived completion
summary based on defined fields, with missing-field keys for translated prompts.

PATCH arrays replace their corresponding owned collections atomically; missing
fields are unchanged. Server validates all entry IDs belong to the caller.
Reject incoherent dates. Never accept `user_id` changes.

Proposed initial resume storage: private persistent volume attached only to the
Go service, plus metadata in PostgreSQL behind a storage interface. This permits
later object storage without changing UI contracts. Production requires a
backed-up persistent volume; container-local ephemeral storage is unacceptable.
Horizontal deployment requires shared/object storage before launch.

MVP accepts PDF up to 10 MiB, checks content type/signature and filename safely,
uses opaque storage keys, streams through authorized Go endpoints, and does not
serve uploads as static public assets. Metadata: `id`, `filename`, `content_type`,
`size_bytes`, `uploaded_at`, `status`. A file is usable only after successful
validation/storage. Replacements do not orphan active metadata on failure.

Applications retain an immutable submitted resume version. Replacing/deleting
the profile resume does not silently change an already submitted application.
The UI explains that distinction before submission/removal. Retention/deletion
policy for those submitted copies must be documented before public launch.

## Saved jobs and applications

| Method / path | Request / response / authorization |
| --- | --- |
| GET `/saved-jobs` | Candidate's paginated saved jobs |
| PUT `/jobs/:id/saved` | Candidate saves visible job, idempotent, 204 |
| DELETE `/jobs/:id/saved` | Candidate removes own save, idempotent, 204 |
| POST `/jobs/:id/applications` | `{resume_id, message?}`; candidate, 201 Application |
| GET `/applications` | Current candidate's applications only |
| PATCH `/applications/:id/withdraw` | Owning candidate; persisted withdrawal |
| GET `/employer/jobs/:id/applications` | Authorized company employer only |
| PATCH `/employer/applications/:id` | `{status}`; authorized vacancy employer only |
| GET `/employer/applications/:id/resume` | Submitted resume for an authorized vacancy only |

Application response: `id`, `job_id`, `company_id`, `candidate_id`, `status`,
`message`, resume metadata reference, `created_at`, `updated_at`, and allowed
summary data. Employer responses include the minimum candidate information
needed for that application, not unrestricted profile access.

Statuses: `sent`, `viewed`, `in_review`, `contacted`, `interview`, `offer`,
`rejected`, `withdrawn`. Candidate cannot assign employer stages. Employer can
advance nonterminal applications or reject them; no silent backward movement.
Withdrawn/rejected are terminal for MVP. Candidate can withdraw a nonterminal
application. Record status changes with actor and timestamp. Contacted denotes
an explicit employer action; do not imply a message-delivery feature exists.

Enforce one application per candidate/job with a database constraint. Duplicate
submission returns 409 with the existing application ID; double clicks cannot
create duplicates. Resume must belong to the candidate and be ready. Job/company
visibility and ownership checks occur transactionally with writes. Concurrent
moderation/closure cannot allow an application to a job already unavailable at
commit under the defined transaction ordering.

Update/invalidate relevant TanStack Query keys only after actual server success.
Saving uses optimistic cache updates with rollback and accessible error toast.
Candidate application lists show an unavailable-job label for moderated content;
they retain the candidate's own application history without exposing hidden text.

## Admin moderation

| Method / path | Request / response / authorization |
| --- | --- |
| GET `/admin/jobs` | Paginated vacancies with lifecycle and moderation filters; admin only |
| POST `/admin/jobs/:id/moderation` | `{action, reason, note?}`; admin, persisted job + audit event |
| GET `/admin/jobs/:id/moderation` | Ordered history; admin only |
| GET `/admin/companies` | Company status and verification; admin only |
| GET `/admin/users` | Minimal permitted user/account metadata; admin only |
| PATCH `/admin/users/:id/status` | `{status, reason, note?}`; suspend/restore employer only in MVP 1 |

Actions: `mark_pending`, `hide`, `reject`, `remove`, `restore`.
Reasons: `fake_vacancy`, `scam`, `duplicate`, `misleading_information`,
`already_closed`, `prohibited_content`, `other`; other requires a note.
Restore requires an explanatory note and never republishes a paused/closed job.
Store actor, target, old/new state, reason, note, timestamp in the same transaction.
Suspension prevents writes and excludes employer vacancies from public results;
restoration does not override individual job moderation states.

MVP 1 admins moderate existing vacancies directly. Reporting queues and review
moderation arrive in later milestones; do not display fake reports. Admin role
alone does not grant applicant/resume access; that requires a separately reviewed
privacy policy and capability.

## Persistence model and migration plan

Add new numbered migrations; do not rewrite the deployed baseline. Proposed
tables in `jobhub`:

- `users`, `sessions`: unique normalized email, password hash, role/status enums,
  hashed session token, expiry, CSRF material and revocation.
- `companies`, `company_memberships`: authorized ownership and public company data.
- `jobs`, `job_skills`: company ownership, structured content, salary, publication,
  moderation, expiration and soft deletion.
- `candidate_profiles`, `work_experience`, `education`, `skills`,
  `candidate_skills`, `candidate_languages`: private structured candidate data.
- `resumes`: private versioned storage metadata and active reference.
- `saved_jobs`: unique candidate/job pair.
- `applications`, `application_status_events`: unique candidate/job pair,
  immutable submitted resume reference, stage history.
- `moderation_events`, `account_status_events`: immutable audit history.

Use foreign keys, check constraints, membership and ownership indexes, public
search indexes, and unique constraints for concurrency-sensitive invariants.
Compute counts from persisted rows, never seeded metrics. Enable RLS for every
new table. Keep `jobhub` outside the Supabase Data API; grant no anon/browser
access. Define a server-only database role and SQL permissions deliberately so
RLS does not accidentally deny all backend queries; end-user authorization
remains enforced by Go and scoped SQL. Migration credentials stay server-side.

## Implementation order and completion gates

1. Approve these proposed endpoints, auth/session policy and private resume storage.
   Then formalize request/response schemas as a versioned OpenAPI contract.
2. Implement auth, company ownership and migrations with negative authorization
   tests before connecting role navigation and short register/login forms.
3. Implement employer vacancy create/edit/publish plus public search/detail and
   company read endpoints; connect forms, URL state, loading/error/empty UI.
4. Implement candidate profile/resume, saves and application submission; connect
   guest return-context flow, candidate applications and employer applicants.
5. Implement admin moderation/history and suspension, and prove public visibility
   is consistent across every query surface after moderation.
6. Run complete browser → Go → PostgreSQL flows and capture persisted evidence.
   Only then mark MVP 1 complete. Later milestones remain explicitly unavailable.

Production builds must use HTTP repositories and exclude development mocks.
Development may explicitly opt into mocks for missing features. Never switch to
mock data after network/auth/server errors. Empty production databases show an
honest empty state and employer creation path. Sample fixtures belong only in
isolated tests or explicitly identified development environments.

Acceptance tests use an isolated Docker database, not production Supabase:

- Employer A registers, creates company and vacancy through UI, publishes; the
  job survives API restart and appears in anonymous search and company vacancies.
- Guest search URL reconstructs filters after reload and Back/Forward. Public
  browsing needs no account. Pagination/filter results match persisted rows.
- Candidate registers through Apply and returns to the same job; uploads resume,
  submits, sees own application; employer A sees the submitted application/resume.
- Employer B cannot edit A's company/job or access applicants/resumes, including
  by manually crafted requests. Candidates cannot create jobs or become admins.
- Private profiles, sessions and resumes are never exposed by public endpoints.
- Duplicate applications, invalid salary/date/enum values, forbidden role changes,
  oversized files, expired sessions and CSRF failures are rejected authoritatively.
- Admin hides/removes the vacancy; fresh search, direct detail, company jobs and
  counts exclude it; applications reject it; moderation survives restart. Owner
  editing cannot clear the restriction. Restore honors publication/expiration.
- Existing health/local migration flow still works; `go test`, `go vet`, frontend
  lint/typecheck/build, integration tests and responsive keyboard/browser QA pass.

Existing unit tests alone do not satisfy these acceptance checks. No real
credential is needed in source code or in a browser environment variable.
