# Telegram vacancy collector

`@jhubkz_bot` collects new vacancy posts from explicitly allowlisted JobHub-owned Telegram channels. It is a separate long-polling process at `backend/cmd/telegram-collector`; it does not provide candidate search or alerts and cannot affect the HTTP API process if it stops.

## Flow and safety boundary

The collector calls the official Telegram Bot API directly with the Go standard library. Startup verifies `getMe.username == jhubkz_bot`. It requests only `channel_post` and `edited_channel_post`, maps the Bot API response into a small raw post model, checks the numeric channel ID allowlist, parses the post deterministically, and sends one normalized external vacancy through the existing ingestion service and jobs upsert.

The server-only configuration is:

- `TELEGRAM_BOT_TOKEN`: required bot token; never expose it as a `VITE_*` variable.
- `TELEGRAM_ALLOWED_CHANNEL_IDS`: comma-separated negative numeric channel IDs. An empty value runs only the `getMe` smoke check and exits.
- `TELEGRAM_DEV_DATABASE_URL`: an explicit loopback `jobhub_telegram_poc_*` database. Persistence is refused outside `APP_ENV=development`.
- `TELEGRAM_OFFSET_FILE`: persistent private checkpoint file, default `./storage/telegram-collector.offset`.

Display names and public usernames never authorize a source. The collector does not discover channels, read history, download media, scrape pages, or run OCR/AI. On first startup it confirms Telegram's pending backlog without ingesting it, then handles only newly arriving posts. Keep the offset file on persistent storage. It is advanced atomically only after an update is safely skipped or its database transaction succeeds. Re-delivery is harmless because the jobs table retains `UNIQUE(source, external_id)`.

## Identity and original links

For channel ID `-100123` and message ID `42`:

- source: `telegram:-100123`
- external ID: `telegram:-100123:42`
- original URL, only when Telegram supplies a public username: `https://t.me/<username>/42`

Channels without a public username are skipped with `missing_source_url`; the collector never invents a link. Telegram jobs use the existing external application behavior, so `apply_url` is the original post and JobHub internal Apply remains unavailable. No fake employer or company account is created.

## Parser and publication rule

The full original text or media caption becomes the description. The parser accepts the first explicitly labelled value for these Russian/Kazakh labels:

- title: `Вакансия`, `Должность`, `Лауазым`, `Бос орын`
- company: `Компания`, `Работодатель`, `Жұмыс беруші`
- location: `Город`, `Место`, `Локация`, `Қала`, `Орналасқан жері`
- salary: `Зарплата`, `Оклад`, `Жалақы`, `Еңбекақы`

It does not translate or infer values. Missing company, location, salary, employment type, work mode, currency, and experience remain NULL. A post is published only with nonempty source text/caption, a meaningful labelled title, stable message identity, and a valid public `t.me` URL. Structured skip reasons include `source_not_allowed`, `empty_text`, `missing_source_url`, `missing_title`, and `validation_failed`. Links embedded in the description remain source text; the external application link deliberately stays the original Telegram post.

Telegram is treated as an event stream. Absence never expires a vacancy. An edited channel post updates the same source/external ID.

## Attach a JobHub test channel

1. Create or open a JobHub-owned **public** test channel and set its public username.
2. Add `@jhubkz_bot` as a channel administrator. Disable posting, editing, deleting, inviting, subscriber management, and other management permissions; it only needs to receive channel posts.
3. Resolve the public channel once through the official Bot API. From the repository root run `set -a; . ./.env; set +a`, then `cd backend` and `go run ./cmd/telegram-collector --resolve-channel <public_username>`. Use the `chat_id` from the safe log output. The token stays in `.env`; do not send it to a lookup bot or website. This username lookup is only setup assistance and never authorizes ingestion.
4. Put the returned negative numeric ID in the root `.env` as `TELEGRAM_ALLOWED_CHANNEL_IDS=-100…`. Keep the existing `TELEGRAM_BOT_TOKEN` unchanged.
5. Create an isolated local database named with the `jobhub_telegram_poc_` prefix, apply the existing local migrations to it, and set its loopback URL in `TELEGRAM_DEV_DATABASE_URL`.
6. From the repository root, load the existing environment and start the collector:

   ```sh
   set -a; . ./.env; set +a
   cd backend
   go run ./cmd/telegram-collector
   ```

7. Publish a new test post after the collector starts, for example:

   ```text
   Вакансия: Go-разработчик
   Компания: JobHub Test
   Город: Алматы
   Зарплата: 500 000 ₸

   Тестовая внешняя вакансия.
   ```

8. Confirm the accepted log contains only the chat/message identity and counts, then inspect the job through the existing public jobs API. Stop the collector with Ctrl-C.

The collector registers only unapproved development sources. Production/staging persistence is blocked, and no production source permission is enabled. Redistribution, storage, commercial-use, attribution, moderation, and vacancy lifecycle policy still require approval before production use.
