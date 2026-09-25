package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jobhub-ai/backend/internal/config"
	"jobhub-ai/backend/internal/database"
	"jobhub-ai/backend/internal/jobs"
	telegramprovider "jobhub-ai/backend/internal/providers/telegram"
	"jobhub-ai/backend/internal/telegramcollector"
	"jobhub-ai/backend/internal/webscanner"
)

const expectedBotUsername = "jhubkz_bot"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, logger, os.Args[1:]); err != nil {
		logger.Error("telegram collector stopped", "category", safeCategory(err))
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger, args []string) error {
	cfg, err := config.LoadTelegram()
	if err != nil {
		return err
	}
	client, err := telegramprovider.NewClient(cfg.Token, &http.Client{})
	if err != nil {
		return err
	}
	checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	bot, err := client.GetMe(checkCtx)
	cancel()
	if err != nil {
		return errors.New("Telegram bot authentication check failed: " + safeCategory(err))
	}
	if bot.Username != expectedBotUsername {
		return errors.New("Telegram bot identity mismatch")
	}
	logger.Info("telegram bot identity verified", "bot_username", bot.Username)
	if len(args) > 0 {
		if len(args) != 2 || args[0] != "--resolve-channel" {
			return errors.New("usage: telegram-collector [--resolve-channel <public_username>]")
		}
		chat, err := client.GetChannel(ctx, args[1])
		if err != nil {
			return errors.New("Telegram channel lookup failed: " + safeCategory(err))
		}
		logger.Info("telegram channel resolved", "chat_id", chat.ID, "channel_username", chat.Username)
		return nil
	}
	if len(cfg.Allowed) == 0 && len(cfg.Admins) == 0 {
		logger.Info("telegram collector smoke check complete", "reason", "no_allowed_channels")
		return nil
	}
	var store *jobs.PostgresStore
	if len(cfg.Allowed) > 0 {
		if err := config.ValidateTelegramDevelopmentDatabase(os.Getenv("APP_ENV"), cfg.DatabaseURL); err != nil {
			return err
		}
		pool, connectErr := database.Connect(ctx, cfg.DatabaseURL, envValue("PGX_QUERY_EXEC_MODE", "cache_statement"))
		if connectErr != nil {
			return errors.New("Telegram development database connection failed")
		}
		defer pool.Close()
		store = jobs.NewPostgresStore(pool)
	}
	processor := telegramcollector.NewProcessor(cfg.Allowed, store, store, logger)
	var websiteScanner telegramcollector.WebsiteScanner = webscanner.New(&http.Client{})
	if os.Getenv("WEBSITE_SCANNER_BROWSER_ENABLED") == "true" {
		websiteScanner = webscanner.NewWithBrowser(&http.Client{}, webscanner.NewChromiumRenderer(os.Getenv("WEBSITE_SCANNER_CHROME_PATH")))
	}
	commands := telegramcollector.NewCommandProcessor(cfg.Admins, websiteScanner, client, logger)
	offset, err := telegramprovider.LoadOffset(cfg.OffsetFile)
	if err != nil {
		return err
	}
	if offset == 0 {
		pending, err := client.LatestPending(ctx)
		if err != nil {
			return errors.New("Telegram offset initialization failed: " + safeCategory(err))
		}
		if len(pending) > 0 {
			offset = pending[len(pending)-1].ID + 1
			if err := telegramprovider.SaveOffset(cfg.OffsetFile, offset); err != nil {
				return err
			}
		}
		logger.Info("telegram update baseline initialized", "pending_updates_skipped", len(pending))
	}
	logger.Info("telegram collector started", "bot_username", bot.Username, "allowed_channel_count", len(cfg.Allowed), "allowed_admin_count", len(cfg.Admins))
	backoff := time.Second
	for {
		updates, err := client.GetUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			wait := backoff
			var apiErr *telegramprovider.APIError
			if errors.As(err, &apiErr) && apiErr.RetryAfter > wait {
				wait = apiErr.RetryAfter
			}
			logger.Warn("telegram polling failed", "category", safeCategory(err), "retry_in_seconds", int(wait.Seconds()))
			if !waitFor(ctx, wait) {
				return nil
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		for _, update := range updates {
			if message, ok := update.DirectMessage(); ok {
				handled, commandErr := commands.Process(ctx, message.ChatID, message.UserID, message.Text, message.LanguageCode)
				if commandErr != nil {
					return errors.New("Telegram command processing failed: " + safeCategory(commandErr))
				}
				if handled {
					offset = update.ID + 1
					if err := telegramprovider.SaveOffset(cfg.OffsetFile, offset); err != nil {
						return err
					}
					continue
				}
			}
			outcome, err := processor.Process(ctx, update)
			if err != nil {
				return errors.New("Telegram update processing failed")
			}
			if outcome.Handled {
				logger.Info("telegram channel post received", "chat_id", outcome.ChatID, "message_id", outcome.MessageID)
				if outcome.Accepted {
					logger.Info("telegram vacancy accepted", "external_id", outcome.ExternalID, "inserted", outcome.Report.Stats.InsertedCount, "updated", outcome.Report.Stats.UpdatedCount)
				} else {
					logger.Info("telegram vacancy skipped", "reason", outcome.SkipReason)
				}
			}
			offset = update.ID + 1
			if err := telegramprovider.SaveOffset(cfg.OffsetFile, offset); err != nil {
				return err
			}
		}
	}
}

func safeCategory(err error) string {
	var apiErr *telegramprovider.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Category
	}
	return err.Error()
}

func waitFor(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func envValue(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
