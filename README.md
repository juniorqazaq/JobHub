# JobHub AI

JobHub AI — веб-приложение для поиска вакансий в Казахстане.

Текущий этап: **MVP 0**. Проект умеет импортировать ограниченное количество вакансий из Jooble Kazakhstan, сохранять их в PostgreSQL, не создавать дубликаты и показывать реальные вакансии во frontend-маркетплейсе.

Jooble сейчас используется только как proof of concept. Для production его нельзя считать одобренным источником, пока не подтверждены права на хранение, показ, переиспользование, удаление и атрибуцию вакансий.

## Стек

- Frontend: React, TypeScript, Vite, TanStack Query, React Router, i18next, React Hook Form, Zod, Lucide.
- Backend: Go, Gin, pgx, sqlc, golang-migrate.
- База данных: PostgreSQL локально через Docker, Supabase Postgres для staging/production.
- Импорт вакансий: server-side Jooble importer. API-ключи не попадают в браузер.

## Что уже есть

- Публичный API:
  - `GET /api/v1/health`
  - `GET /api/v1/jobs`
  - `GET /api/v1/jobs/:id`
- Схема БД с поддержкой разных источников вакансий.
- Дедупликация импортированных вакансий через `UNIQUE(source, external_id)`.
- Атомарный импорт с логированием статистики.
- Внешние вакансии открываются только через external CTA.
- Frontend: главная страница, список вакансий, детальная страница, поиск, сортировка, loading/empty/error states.
- i18n: казахский, русский, английский.

## Быстрый запуск локально

Нужно установить:

- Docker + Docker Compose v2
- Go 1.25+
- Node.js 22.12+ и npm

Создать локальный `.env`:

```sh
cp .env.example .env
```

Запустить PostgreSQL, миграции и backend:

```sh
make local-up
```

Запустить frontend:

```sh
cd frontend
npm ci
npm run dev
```

Открыть:

- Frontend: http://localhost:5173
- Backend health check: http://localhost:8080/api/v1/health

Остановить локальные сервисы:

```sh
make local-down
```

## Запуск backend вручную

Если нужно запустить Go backend без Docker-контейнера backend:

```sh
docker compose --profile local up -d postgres
make migrate-local

set -a
. ./.env
set +a

cd backend
go run ./cmd/api
```

Не запускайте Docker backend и ручной Go backend на одном порту одновременно.

## Импорт вакансий Jooble

Реальный ключ хранится только в локальном `.env`:

```dotenv
JOOBLE_API_KEY=<your-jooble-kazakhstan-api-key>
```

Запуск импорта:

```sh
make migrate-local
make import-jooble
```

Импорт ограничен для POC. `JOOBLE_MAX_REQUESTS` не может быть больше 50. API-ключ не логируется и не должен попадать в git.

Проверить вакансии:

```sh
curl 'http://localhost:8080/api/v1/jobs?page=1&page_size=20'
```

## Supabase

Для staging/production backend подключается к Supabase Postgres через переменные окружения. Go backend остается основным backend-сервисом.

Пример только с placeholder-значениями:

```dotenv
APP_ENV=staging
DATABASE_URL=postgresql://postgres.<project-ref>:<PASSWORD>@<host>:5432/postgres?sslmode=require&search_path=public
DATABASE_MIGRATION_URL=postgresql://postgres.<project-ref>:<PASSWORD>@<host>:5432/postgres?sslmode=require&search_path=public
PGX_QUERY_EXEC_MODE=cache_statement
```

Важно:

- Для Supabase используйте `sslmode=require`.
- Для локального Docker Postgres используйте `sslmode=disable`.
- Миграции запускайте через direct или session connection.
- Не запускайте миграции через transaction pooler на порту `6543`.
- Если приложение использует transaction pooler, установите `PGX_QUERY_EXEC_MODE=exec`.
- Не открывайте схему `jobhub` через Supabase Data API или anon key.

Запуск remote migrations:

```sh
make migrate-staging
```

## Переменные окружения

| Variable | Для чего |
| --- | --- |
| `DATABASE_URL` | Основное подключение backend к PostgreSQL |
| `DATABASE_MIGRATION_URL` | Отдельное подключение для миграций |
| `PGX_QUERY_EXEC_MODE` | Режим pgx, обычно `cache_statement` или `exec` |
| `FRONTEND_ORIGIN` | Разрешенный frontend origin для CORS |
| `VITE_API_URL` | URL backend API для frontend |
| `VITE_USE_MOCKS` | Включает mock-данные во frontend |
| `JOOBLE_API_KEY` | Server-side ключ Jooble |
| `JOOBLE_MAX_REQUESTS` | Лимит запросов импорта, максимум 50 |
| `JOOBLE_SEARCHES` | Поисковые запросы в формате `keywords|location` |

Никогда не коммитьте `.env`, API-ключи, пароли от базы, Supabase keys или connection strings с реальными credentials.

## Проверки

Backend:

```sh
cd backend
go test -race ./...
go vet ./...
go build ./...
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate
```

Frontend:

```sh
cd frontend
npm run lint
npm run typecheck
npm run build
```

Repository:

```sh
git diff --check
git status
```

## Чего пока нет

Это отложено на следующие этапы:

- авторизация
- профили пользователей
- резюме
- отклики
- кабинет работодателя
- админ-модерация
- production-разрешение на использование внешних вакансий

## Безопасность

- `.env` игнорируется git.
- `.env.example` содержит только placeholder-значения.
- Jooble API key хранится только на backend стороне.
- Импортированные вакансии не создают фейковые аккаунты работодателей.
- Внешние вакансии ведут на внешний apply link.
- Production Jooble import отключен, пока не подтверждены права провайдера.
