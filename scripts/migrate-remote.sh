#!/bin/sh
set -eu

action="${1:-version}"
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(dirname "$script_dir")

if [ -z "${DATABASE_MIGRATION_URL:-}" ] && [ -f "$repo_dir/.env" ]; then
	set -a
	# .env is intentionally untracked and may contain the remote connection URLs.
	. "$repo_dir/.env"
	set +a
fi

if [ -z "${DATABASE_MIGRATION_URL:-}" ]; then
	echo "Set DATABASE_MIGRATION_URL before running staging migrations." >&2
	exit 1
fi

cd "$repo_dir/backend"
exec go run ./cmd/migrate --action "$action" --path ./migrations
