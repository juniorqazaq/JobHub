#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(dirname "$script_dir")
database_name=${JOBHUB_POC_DATABASE:-jobhub_ats_poc_careers_20260925}
database_url="postgresql://localhost:5432/${database_name}?sslmode=disable"

for command_name in go npm psql pg_isready lsof; do
	if ! command -v "$command_name" >/dev/null 2>&1; then
		echo "Required command is missing: $command_name" >&2
		exit 1
	fi
done

case "$database_name" in
	*[!A-Za-z0-9_]*)
		echo "JOBHUB_POC_DATABASE must contain only letters, numbers, and underscores." >&2
		exit 1
		;;
esac

if ! pg_isready -h localhost -p 5432 >/dev/null 2>&1; then
	echo "Local PostgreSQL is not available on localhost:5432." >&2
	exit 1
fi

if [ "$(psql -h localhost -d postgres -X -Atqc "SELECT 1 FROM pg_database WHERE datname = '$database_name'")" != "1" ]; then
	echo "Local vacancy database does not exist: $database_name" >&2
	exit 1
fi

if [ "$(psql -h localhost -d "$database_name" -X -Atqc "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='jobhub' AND table_name='jobs' AND column_name='canonical_city_id')")" != "t" ]; then
	echo "Local vacancy database is missing the canonical city schema." >&2
	exit 1
fi

if [ "$(psql -h localhost -d "$database_name" -X -Atqc "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='jobhub' AND table_name='companies' AND column_name='industry')")" != "t" ]; then
	echo "Applying the local company profile schema..."
	psql -h localhost -d "$database_name" -X -v ON_ERROR_STOP=1 -1 \
		-f "$repo_dir/backend/migrations/000010_create_company_profiles_and_follows.up.sql"
fi

if lsof -nP -iTCP:8080 -sTCP:LISTEN >/dev/null 2>&1; then
	echo "Port 8080 is already in use. Stop the existing backend first." >&2
	exit 1
fi

if lsof -nP -iTCP:5173 -sTCP:LISTEN >/dev/null 2>&1; then
	echo "Port 5173 is already in use. Stop the existing frontend first." >&2
	exit 1
fi

if [ -f "$repo_dir/.env" ]; then
	set -a
	# The local file may contain staging credentials; the database and environment
	# are overridden below before either development process is started.
	. "$repo_dir/.env"
	set +a
fi

export APP_ENV=development
export DATABASE_URL=$database_url
export FRONTEND_ORIGIN=http://localhost:5173
export VITE_API_URL=http://localhost:8080/api/v1
export PGX_QUERY_EXEC_MODE=cache_statement

job_count=$(psql -h localhost -d "$database_name" -X -Atqc "SELECT count(*) FROM jobhub.jobs j WHERE jobhub.job_is_public(j)")
echo "Starting JobHub with $job_count visible vacancies from $database_name."
echo "Frontend: http://localhost:5173"
echo "Backend:  http://localhost:8080"

backend_pid=
frontend_pid=

cleanup() {
	trap - INT TERM EXIT
	[ -z "$frontend_pid" ] || kill "$frontend_pid" 2>/dev/null || true
	[ -z "$backend_pid" ] || kill "$backend_pid" 2>/dev/null || true
	wait 2>/dev/null || true
}
trap cleanup INT TERM EXIT

(
	cd "$repo_dir/backend"
	exec go run ./cmd/api
) &
backend_pid=$!

(
	cd "$repo_dir/frontend"
	if [ ! -d node_modules ]; then
		npm ci
	fi
	exec npm run dev
) &
frontend_pid=$!

while kill -0 "$backend_pid" 2>/dev/null && kill -0 "$frontend_pid" 2>/dev/null; do
	sleep 1
done

echo "A JobHub development process stopped unexpectedly." >&2
exit 1
