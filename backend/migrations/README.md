# Migration security rules

The `jobhub` schema has no application tables in Phase 1.5. Every future
migration that creates a table must enable Row Level Security in the same `up`
migration, even though the Go backend normally connects with a trusted
server-side database role:

```sql
CREATE TABLE jobhub.example (...);
ALTER TABLE jobhub.example ENABLE ROW LEVEL SECURITY;
```

Do not grant `anon` or `authenticated` access to the `jobhub` schema or add it
to the Supabase Data API's exposed schemas until Phase 2 defines explicit RLS
policies. RLS without a policy denies Data API access while database owners and
roles with `BYPASSRLS` retain server-side access. Add ownership-based policies
alongside any future client access; never use a permissive catch-all policy.

Migration `000002_create_vacancy_supply` follows this rule for source-aware jobs,
company-source identities and ingestion runs. Its seeded `jooble:kz` source keeps
`production_permissions_confirmed=false`; do not change that flag until the
provider's API-specific storage, redistribution, retention, attribution and
removal permissions are confirmed.
