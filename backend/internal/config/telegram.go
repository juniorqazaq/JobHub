package config

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type TelegramConfig struct {
	Token       string
	Allowed     map[int64]struct{}
	OffsetFile  string
	DatabaseURL string
}

func LoadTelegram() (TelegramConfig, error) {
	c := TelegramConfig{
		Token: strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")), OffsetFile: value("TELEGRAM_OFFSET_FILE", "./storage/telegram-collector.offset"),
		DatabaseURL: strings.TrimSpace(os.Getenv("TELEGRAM_DEV_DATABASE_URL")), Allowed: map[int64]struct{}{},
	}
	if c.Token == "" {
		return TelegramConfig{}, errors.New("TELEGRAM_BOT_TOKEN is required")
	}
	if strings.TrimSpace(c.OffsetFile) == "" || filepath.Clean(c.OffsetFile) == "." {
		return TelegramConfig{}, errors.New("TELEGRAM_OFFSET_FILE must name a file")
	}
	raw := strings.TrimSpace(os.Getenv("TELEGRAM_ALLOWED_CHANNEL_IDS"))
	if raw == "" {
		return c, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) > 20 {
		return TelegramConfig{}, errors.New("TELEGRAM_ALLOWED_CHANNEL_IDS exceeds 20 channels")
	}
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id >= 0 {
			return TelegramConfig{}, errors.New("TELEGRAM_ALLOWED_CHANNEL_IDS must contain negative numeric channel IDs")
		}
		if _, duplicate := c.Allowed[id]; duplicate {
			return TelegramConfig{}, errors.New("TELEGRAM_ALLOWED_CHANNEL_IDS contains a duplicate")
		}
		c.Allowed[id] = struct{}{}
	}
	return c, nil
}

func ValidateTelegramDevelopmentDatabase(environment, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || environment != "development" || (u.Scheme != "postgres" && u.Scheme != "postgresql") || (u.Hostname() != "127.0.0.1" && u.Hostname() != "::1") || u.Port() == "" || u.Fragment != "" || !regexp.MustCompile(`^/jobhub_telegram_poc_[A-Za-z0-9_]+$`).MatchString(u.Path) {
		return errors.New("Telegram ingestion requires APP_ENV=development and an explicit loopback jobhub_telegram_poc_* database")
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return errors.New("invalid development database options")
	}
	for key := range q {
		if key != "sslmode" {
			return errors.New("database connection overrides are not allowed")
		}
	}
	return nil
}
