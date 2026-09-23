#!/bin/sh
set -eu

action="${1:-up}"
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(dirname "$script_dir")

if [ -z "${DATABASE_MIGRATION_URL:-}" ] && [ -z "${DATABASE_URL:-}" ] && [ -f "$repo_dir/.env" ]; then
	set -a
	# .env is intentionally untracked and may contain the remote connection URLs.
	. "$repo_dir/.env"
	set +a
fi

migration_url="${DATABASE_MIGRATION_URL:-${DATABASE_URL:-}}"

if [ -z "$migration_url" ]; then
	echo "Set DATABASE_MIGRATION_URL (preferred) or DATABASE_URL before running remote migrations." >&2
	exit 1
fi

case "$migration_url" in
	*:6543/*|*:6543\?*)
		echo "Refusing to run migrations through the transaction pooler on port 6543. Use a direct or session-mode URL on port 5432." >&2
		exit 1
		;;
esac

docker run --rm \
	-v "$repo_dir/backend/migrations:/migrations:ro" \
	migrate/migrate:v4.20.1 \
	-path /migrations -database "$migration_url" "$action"
