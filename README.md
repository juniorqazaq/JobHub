# JobHub AI

JobHub AI — веб-приложение для поиска вакансий в Казахстане.

Текущий этап: **MVP 1C**. Проект поддерживает авторизацию, вакансии JobHub и Jooble, профиль соискателя, приватное PDF-резюме, сохраненные вакансии и полный цикл отклика на собственные вакансии JobHub.

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
  - `GET /api/v1/companies`
  - `GET /api/v1/companies/:id`
- Auth API:
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/login`
  - `GET /api/v1/auth/me`
  - `POST /api/v1/auth/logout`
- Employer vacancy API:
  - `GET /api/v1/employer/jobs`
  - `GET /api/v1/employer/jobs/:id`
  - `POST /api/v1/jobs`
  - `PATCH /api/v1/jobs/:id`
  - `DELETE /api/v1/jobs/:id`
  - `GET /api/v1/employer/company`, `PATCH /api/v1/employer/company`
- Candidate API:
  - `GET /api/v1/profile`, `PATCH /api/v1/profile`
  - `POST /api/v1/profile/resume`, `GET /api/v1/profile/resume`, `GET /api/v1/profile/resume/content`, `DELETE /api/v1/profile/resume`
  - `GET /api/v1/saved-jobs`, `PUT /api/v1/jobs/:id/saved`, `DELETE /api/v1/jobs/:id/saved`
  - `GET /api/v1/companies/:id/follow`, `PUT /api/v1/companies/:id/follow`, `DELETE /api/v1/companies/:id/follow`
  - `POST /api/v1/jobs/:id/applications`, `GET /api/v1/applications`, `PATCH /api/v1/applications/:id/withdraw`
- Employer applicant API:
  - `GET /api/v1/employer/jobs/:id/applications`
  - `GET /api/v1/employer/applications/:id/resume`
  - `PATCH /api/v1/employer/applications/:id`
- Схема БД с поддержкой разных источников вакансий.
- Регистрация соискателя и работодателя.
- При регистрации работодателя создается компания и ownership membership.
- Сессии через HttpOnly cookie, CSRF header для mutations, session token хранится в БД только как hash.
- Дедупликация импортированных вакансий через `UNIQUE(source, external_id)`.
- Атомарный импорт с логированием статистики.
- Внешние вакансии открываются только через external CTA.
- Frontend: главная страница, список вакансий, детальная страница, поиск, сортировка, loading/empty/error states.
- Кабинет работодателя: список вакансий, создание, редактирование, публикация, пауза, закрытие и soft delete.
- Собственные вакансии проходят lifecycle `draft → published ↔ paused → closed`; `closed` нельзя опубликовать повторно.
- Структурированный профиль соискателя: опыт, образование, навыки, языки, предпочтения и серверный процент заполнения.
- Приватное PDF-резюме хранится вне публичной директории; доступ проверяет Go backend. Максимальный размер — 10 МБ.
- Сохранение вакансий идемпотентно. На одну вакансию разрешен один отклик одного соискателя.
- Работодатель видит отклики только на вакансии своей компании и переводит их вперед по этапам `sent → viewed → in_review → contacted → interview → offer` или отклоняет.
- Отклик сохраняет конкретную версию резюме. Замена активного резюме не меняет уже отправленный отклик.
- Внутренний отклик доступен только для опубликованных собственных вакансий JobHub; Jooble сохраняет внешний CTA.
- Публичный каталог и профили компаний с открытыми вакансиями, verified-статусом и подпиской соискателя.
- Работодатель может редактировать публичный профиль своей компании; ownership проверяет backend.
- i18n: казахский и русский. Архитектура ключей сохраняет возможность добавить английский позже.

## Быстрый запуск локально

Нужно установить:

- Docker + Docker Compose v2
- Go 1.25+
- Node.js 22.12+ и npm

Создать локальный `.env`:

```sh
cp .env.example .env
```

### Локальный запуск с собранными вакансиями

Если локальная POC-база `jobhub_ats_poc_careers_20260925` уже создана, весь
проект запускается одной командой без Docker:

```sh
make dev
```

Команда всегда запускает backend в `development`-режиме с локальной POC-базой,
поэтому вакансии Kcell, Air Astana, Technodom, Halyk и Kolesa видны в интерфейсе.
Она не использует staging Supabase как локальную базу и не меняет данные Supabase.
Название POC-базы можно переопределить через `JOBHUB_POC_DATABASE`.

Остановить оба процесса можно через `Ctrl+C`.

### Docker-запуск с пустой локальной базой

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
make migrate-version
make migrate-staging
```

Команды используют локальный Go runner и не требуют Docker. Для staging
обязательно задайте `APP_ENV=staging` и `DATABASE_MIGRATION_URL` с direct или
session подключением на порту `5432`. `make migrate-version` безопасно показывает
текущую версию, dirty-состояние и ожидающие версии. Production, transaction
pooler, `sslmode=disable`, автоматические down/reset/force операции запрещены.

## Admin provisioning

Admin не регистрируется публично. Создать первого admin можно только backend-командой:

```sh
set -a
. ./.env
set +a

export ADMIN_PASSWORD='<strong-admin-password>'
cd backend
go run ./cmd/create-admin --full-name 'Admin Name' --email admin@example.com
```

Пароль не печатается и не хранится в репозитории.

## Проверка employer flow

1. Зарегистрируйтесь как работодатель и укажите компанию.
2. Откройте `/employer/vacancies/new`.
3. Сохраните черновик или сразу опубликуйте вакансию.
4. Проверьте публикацию в `/jobs` и `/jobs/:id`.
5. Через `/employer/vacancies` приостановите, повторно опубликуйте или закройте вакансию.

Черновики, приостановленные, закрытые, удаленные и просроченные вакансии не показываются публично.

## Проверка candidate flow

1. Зарегистрируйтесь как соискатель и заполните `/profile`.
2. Загрузите PDF-резюме размером не более 10 МБ.
3. Сохраните вакансию JobHub или Jooble на странице вакансии.
4. На опубликованной вакансии JobHub отправьте отклик и проверьте его в `/applications`.
5. Войдите как работодатель и откройте кандидатов через `/employer/vacancies/:jobId/applicants`.

Для Docker резюме сохраняются в приватном volume `resume_data`. При ручном запуске путь задает `RESUME_STORAGE_DIR`; директорию нужно подключить к постоянному приватному диску в production.

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
| `RESUME_STORAGE_DIR` | Приватная директория для PDF-резюме |

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

## Ограничения текущего этапа

Это отложено на следующие этапы:

- админ-модерация
- уведомления, чат и публичный каталог кандидатов
- облачное object storage для резюме; сейчас используется приватный persistent volume
- production-разрешение на использование внешних вакансий

## Безопасность

- `.env` игнорируется git.
- `.env.example` содержит только placeholder-значения.
- Jooble API key хранится только на backend стороне.
- Импортированные вакансии не создают фейковые аккаунты работодателей.
- Внешние вакансии ведут на внешний apply link.
- Production Jooble import отключен, пока не подтверждены права провайдера.
