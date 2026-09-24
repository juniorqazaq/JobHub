
<!-- jobhub-focused-change:begin -->
## Focused repository work

- For scoped JobHub fixes, use the `jobhub-focused-change` skill when available.
- Use `docs/agent-repo-map.md` as a locator when present. Verify relevant paths and code; the map is not an implementation snapshot.
- Start with the affected feature. Do not repeat a whole-repository audit unless requested or required by a concrete dependency.
- Prefer scoped symbol searches and complete relevant functions/queries over repeated whole-file reads.
- Re-read when content changed, was truncated, is no longer available, or a contract/dependency needs verification. Necessary reads and safety checks take priority over speed.
- Treat security, ownership, migrations and shared contracts as cross-layer work when needed. Do not hide bugs by avoiding relevant code.
- Keep the repository map short; update only verified entries affected by structural changes.
- Do not print secrets or private files, use staging/production for destructive tests, or consume provider quota without authorization.
- Preserve unrelated work and `roadmap.md`. Do not commit, push, or change global Codex settings without authorization.

## Current UI locales

- JobHub currently supports only `kk` and `ru` in the active UI.
- English is deferred for a later phase.
- For new UI features, add or update Kazakh and Russian translation keys only.
- Do not create English translation files, add English to locale selectors, or run English browser QA for the current MVP.
- Keep the i18n key architecture extensible so English can be restored later by adding an English locale file, registering `en`, and adding it to the selector.
<!-- jobhub-focused-change:end -->
