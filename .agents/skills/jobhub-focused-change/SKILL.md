---
name: jobhub-focused-change
description: "Make scoped JobHub bug fixes and small feature changes with targeted repository navigation, minimal relevant reads, and regression checks. Use for repeated search/read loops, UI overflow, DTO mapping, and existing marketplace flows. Not a substitute for an explicitly requested full audit, security review, or architectural redesign."
---

# JobHub focused change

## Goal

Solve the requested problem without rediscovering the whole repository. Reduce redundant reads, not necessary verification. This is an instruction-only workflow, not a file cache or persistent memory system.

Follow applicable AGENTS.md instructions, task requirements, and relevant installed skills. Never skip a required security check or broaden permissions to save time.

## 1. Establish the task boundary

- Identify the requested behavior, observable failure, acceptance criteria, and excluded work.
- Use screenshots/logs as evidence, not as permission to invent missing implementation details.
- Check the current worktree/branch and changed filenames once with `git status --short --branch` and `git diff --name-only`.
- Preserve unrelated user changes, including an existing untracked `roadmap.md`.
- Use `docs/agent-repo-map.md` when it exists. Read only the relevant section if it is large. Treat it as a locator, not authoritative code or proof a feature is complete.
- Verify that mapped paths exist. A previous conversation or map may describe another revision/worktree.

## 2. Navigate before searching broadly

Start at the feature boundary implicated by the task:

| Task | First inspection | Expand only when evidence requires it |
| --- | --- | --- |
| Overflow, spacing, responsive UI | Affected page/component and owning styles; reproduce in browser | Shared layout, primitives, or actual field shape |
| Incorrect displayed data | Component, mapper/domain type, HTTP DTO | Handler/service and the specific SQL query |
| Validation or permissions | Relevant API route, service, authorization and tests | Owning database query/constraint |
| Import normalization | Provider adapter, normalizer and fixtures | Ingestion/store logic |
| Auth request failure | Actual request URL/status, API routing, origin/cookie configuration | Auth service and relevant database state |

Locate files by name before content searching. Prefer exact paths and symbols from the task or map.

- Search in the smallest relevant directory; widen only when the first search fails or reveals a dependency.
- Batch related symbol searches into one command when practical, for example `rg -n -e 'salary_visible' -e 'SalaryVisible' <verified-file-or-directory>`.
- Read a complete relevant function/component/query and nearby types, not arbitrary tiny fragments or repeated entire files.
- A new `Read jobs.sql` event may be a different useful range; do not assume a repeated filename is redundant.
- For sqlc, locate the actual config and query source first. Do not guess generated-code paths or repeatedly search all SQL files.
- Do not preload all migrations, generated code, test fixtures, node_modules, build output, or unrelated feature documentation.
- Once enough evidence exists to make and verify a focused fix, stop discovery and implement it.

If no map exists, locate only what this task needs. Do not turn every bug fix into a repository-indexing task. Create a broader compact map only during an explicitly requested setup/map-maintenance task.

## 3. Reuse context without trusting stale information

Keep a small task-local list of paths, relevant symbols, observed behavior, and unresolved questions. It need not become another committed file.

Do not repeat the same search/read of the same unchanged range while its result remains available. Re-read when:

- the file or worktree changed;
- a needed section was not included or output was truncated;
- earlier content is unavailable after context compaction;
- an exact contract/dependency must be checked;
- the reproduction or tests contradict the current understanding.

There is no hard cap on reads. Correctness, privacy, and authorization take precedence. For a meaningful scope expansion, explain the concrete dependency briefly instead of restarting a generic full audit.

## 4. Preserve JobHub boundaries

These are intended project constraints; verify the current implementation where the task touches them:

- React accesses the Go API through the existing typed repository layer; no browser-to-database/provider-secret access.
- Reuse the existing UI tokens, i18n keys, query caching, and feature implementations.
- Native `source=jobhub` and imported `source=jooble:kz` jobs keep distinct application behavior. Do not turn an external link into an internal application.
- Ownership and private resume/application access remain server-enforced; hidden UI controls are not authorization.
- Respect the existing common public-visibility logic.
- Never manually edit generated sqlc output. Modify the source query/config and regenerate with the project's installed/pinned command when necessary.
- Do not rewrite applied migrations or introduce contract/schema changes for a purely visual fix.
- Do not assume a test passed or an endpoint exists because a previous report said so.

## 5. Make the smallest complete fix

- Reproduce the problem first when the environment permits it.
- Read the code that will be changed; do not edit from remembered line numbers alone.
- Fix the underlying cause, not only the screenshot symptom.
- Keep necessary frontend/backend contract changes coherent; do not artificially restrict a genuinely cross-layer fix to one file.
- Add or update a targeted regression test where practical.
- For long-text UI fixes, check unbroken text, multiline text, multi-word skills, and narrow screens. Do not globally hide overflow or truncate long-form content to conceal a layout bug.
- Avoid unrelated redesigns, dependency upgrades, renames, and new product features.

## 6. Verify proportionally and honestly

Honor the current task's quality gates. During iteration, run focused tests first; before handoff, run the required final checks.

For frontend changes, use the existing package scripts for lint, typecheck and build. For layout changes, verify the affected route in a browser at desktop and mobile widths and compare with the supplied screenshot.

For backend changes, run the affected tests plus the required Go tests, vet/build, and sqlc generation checks. Schema/authorization/storage changes need appropriate isolated database coverage.

- Inspect package.json/Makefile only when the correct command is not already known in this worktree. Do not invent a `make run` or other target.
- Use a deliberately configured isolated test database. Never alias TEST_DATABASE_URL to DATABASE_URL without verifying the target is an approved isolated test database.
- Do not run destructive tests against Supabase staging/production.
- Do not consume Jooble requests, migrate remote databases, or perform other external writes without task authorization.
- Do not print .env contents, connection strings, credentials, session tokens, or private resume contents. Environment diagnostics should disclose presence/configuration safely, not secret values.
- Run `git diff --check` and inspect only the task's final diff.
- Report unavailable tools or skipped checks as blocked/not run, never as passed.

## 7. Keep guidance small

If this task moved a mapped file or changed an important entry point, update only the affected verified entries in docs/agent-repo-map.md when that file exists. Do not copy source code, full schemas, old completion reports, or secrets into the map. Do not update the user's roadmap unless asked.

Commit only when requested. Use logical Conventional Commits with related tests; never rewrite published history or push without authorization. Leave unrelated files unchanged.

## Handoff

Report the cause, changed files, actual checks/results, and any blocker. If a broader investigation was necessary, state why. Keep the report factual and compact; do not dump internal reasoning or every search command.
