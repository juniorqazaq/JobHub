.PHONY: local-up local-down migrate-local migrate-staging

local-up:
	docker compose --profile local up --build

local-down:
	docker compose --profile local down

migrate-local:
	docker compose --profile local run --rm migrate

migrate-staging:
	./scripts/migrate-remote.sh up
