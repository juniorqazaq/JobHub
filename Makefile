.PHONY: local-up local-down migrate-local migrate-staging migrate-version import-jooble

local-up:
	docker compose --profile local up --build

local-down:
	docker compose --profile local down

migrate-local:
	docker compose --profile local run --rm migrate

migrate-staging:
	./scripts/migrate-remote.sh up

migrate-version:
	./scripts/migrate-remote.sh version

import-jooble:
	@set -a; . ./.env; set +a; cd backend && go run ./cmd/import-jooble
