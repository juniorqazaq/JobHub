package telegram

import "time"

// Post is the provider-neutral raw Telegram representation. Bot API response
// structs remain private to the client so parsing is independent of transport.
type Post struct {
	ChatID          int64
	MessageID       int64
	ChannelUsername string
	ChannelTitle    string
	Text            string
	OriginalURL     string
	TelegramDate    time.Time
	ReceivedAt      time.Time
	Edited          bool
}

type SkipReason string

const (
	SkipSourceNotAllowed SkipReason = "source_not_allowed"
	SkipEmptyText        SkipReason = "empty_text"
	SkipMissingSourceURL SkipReason = "missing_source_url"
	SkipMissingTitle     SkipReason = "missing_title"
	SkipValidationFailed SkipReason = "validation_failed"
)

type Parsed struct {
	Title    string
	Company  string
	Location string
	Salary   string
}
