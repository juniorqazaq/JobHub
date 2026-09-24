package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const apiBase = "https://api.telegram.org"
const maxResponseBytes = 4 << 20

var usernamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{4,31}$`)

type Bot struct {
	ID       int64
	Username string
}

type Chat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Username string `json:"username"`
}

type Update struct {
	ID                int64
	Message           *message
	ChannelPost       *message
	EditedChannelPost *message
}

type message struct {
	MessageID int64  `json:"message_id"`
	Date      int64  `json:"date"`
	EditDate  int64  `json:"edit_date"`
	Text      string `json:"text"`
	Caption   string `json:"caption"`
	Chat      struct {
		ID       int64  `json:"id"`
		Type     string `json:"type"`
		Title    string `json:"title"`
		Username string `json:"username"`
	} `json:"chat"`
	From struct {
		ID           int64  `json:"id"`
		LanguageCode string `json:"language_code"`
	} `json:"from"`
}

func (u *Update) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID                int64    `json:"update_id"`
		Message           *message `json:"message"`
		ChannelPost       *message `json:"channel_post"`
		EditedChannelPost *message `json:"edited_channel_post"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	u.ID, u.Message, u.ChannelPost, u.EditedChannelPost = raw.ID, raw.Message, raw.ChannelPost, raw.EditedChannelPost
	return nil
}

type APIError struct {
	Category   string
	RetryAfter time.Duration
}

func (e *APIError) Error() string { return e.Category }

type Client struct {
	token string
	http  *http.Client
	base  string
}

func NewClient(token string, client *http.Client) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("TELEGRAM_BOT_TOKEN is required")
	}
	if client == nil {
		client = &http.Client{}
	}
	copyClient := *client
	copyClient.Timeout = 45 * time.Second
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{token: token, http: &copyClient, base: apiBase}, nil
}

func newTestClient(token, base string, client *http.Client) (*Client, error) {
	c, err := NewClient(token, client)
	if err == nil {
		c.base = strings.TrimRight(base, "/")
	}
	return c, err
}

func (c *Client) GetMe(ctx context.Context) (Bot, error) {
	var bot struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
	}
	if err := c.call(ctx, "getMe", nil, &bot); err != nil {
		return Bot{}, err
	}
	if bot.ID == 0 || !usernamePattern.MatchString(bot.Username) {
		return Bot{}, &APIError{Category: "TELEGRAM_INVALID_RESPONSE"}
	}
	return Bot{ID: bot.ID, Username: bot.Username}, nil
}

// GetChannel resolves one explicitly named public channel for allowlist setup.
// Its username is lookup input only; runtime authorization always uses Chat.ID.
func (c *Client) GetChannel(ctx context.Context, username string) (Chat, error) {
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if !usernamePattern.MatchString(username) {
		return Chat{}, &APIError{Category: "TELEGRAM_INVALID_CHANNEL_USERNAME"}
	}
	var chat Chat
	if err := c.call(ctx, "getChat", struct {
		ChatID string `json:"chat_id"`
	}{"@" + username}, &chat); err != nil {
		return Chat{}, err
	}
	if chat.ID >= 0 || chat.Type != "channel" || !strings.EqualFold(chat.Username, username) {
		return Chat{}, &APIError{Category: "TELEGRAM_INVALID_CHANNEL"}
	}
	return chat, nil
}

func (c *Client) GetUpdates(ctx context.Context, offset int64) ([]Update, error) {
	return c.getUpdates(ctx, offset, 30)
}

// LatestPending performs a nonblocking negative-offset read. Telegram confirms
// all earlier pending updates, allowing a first run to establish a "new only"
// baseline without importing backlog.
func (c *Client) LatestPending(ctx context.Context) ([]Update, error) {
	return c.getUpdates(ctx, -1, 0)
}

func (c *Client) getUpdates(ctx context.Context, offset int64, timeout int) ([]Update, error) {
	payload := struct {
		Offset         int64    `json:"offset"`
		Timeout        int      `json:"timeout"`
		AllowedUpdates []string `json:"allowed_updates"`
	}{offset, timeout, []string{"message", "channel_post", "edited_channel_post"}}
	var updates []Update
	if err := c.call(ctx, "getUpdates", payload, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

type DirectMessage struct {
	ChatID, UserID     int64
	Text, LanguageCode string
}

func (u Update) DirectMessage() (DirectMessage, bool) {
	if u.Message == nil || u.Message.Chat.Type != "private" || u.Message.Chat.ID == 0 || u.Message.From.ID == 0 {
		return DirectMessage{}, false
	}
	return DirectMessage{ChatID: u.Message.Chat.ID, UserID: u.Message.From.ID, Text: strings.TrimSpace(u.Message.Text), LanguageCode: u.Message.From.LanguageCode}, true
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	if chatID == 0 || strings.TrimSpace(text) == "" || len([]rune(text)) > 4096 {
		return &APIError{Category: "TELEGRAM_INVALID_MESSAGE"}
	}
	var sent struct {
		MessageID int64 `json:"message_id"`
	}
	if err := c.call(ctx, "sendMessage", struct {
		ChatID         int64  `json:"chat_id"`
		Text           string `json:"text"`
		DisablePreview bool   `json:"disable_web_page_preview"`
	}{chatID, text, true}, &sent); err != nil {
		return err
	}
	if sent.MessageID == 0 {
		return &APIError{Category: "TELEGRAM_INVALID_RESPONSE"}
	}
	return nil
}

func (c *Client) call(ctx context.Context, method string, payload any, target any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return &APIError{Category: "TELEGRAM_REQUEST_FAILED"}
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/bot"+url.PathEscape(c.token)+"/"+method, body)
	if err != nil {
		return &APIError{Category: "TELEGRAM_REQUEST_FAILED"}
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return &APIError{Category: "TELEGRAM_TRANSPORT_FAILED"}
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
	if err != nil || len(data) > maxResponseBytes {
		return &APIError{Category: "TELEGRAM_RESPONSE_FAILED"}
	}
	var envelope struct {
		OK         bool            `json:"ok"`
		Result     json.RawMessage `json:"result"`
		ErrorCode  int             `json:"error_code"`
		Parameters struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return &APIError{Category: "TELEGRAM_INVALID_RESPONSE"}
	}
	if res.StatusCode != http.StatusOK || !envelope.OK {
		category := "TELEGRAM_API_FAILED"
		if res.StatusCode == http.StatusUnauthorized || envelope.ErrorCode == http.StatusUnauthorized {
			category = "TELEGRAM_AUTH_FAILED"
		} else if res.StatusCode == http.StatusTooManyRequests || envelope.ErrorCode == http.StatusTooManyRequests {
			category = "TELEGRAM_RATE_LIMITED"
		}
		return &APIError{Category: category, RetryAfter: time.Duration(envelope.Parameters.RetryAfter) * time.Second}
	}
	if len(envelope.Result) == 0 || json.Unmarshal(envelope.Result, target) != nil {
		return &APIError{Category: "TELEGRAM_INVALID_RESPONSE"}
	}
	return nil
}

func MapUpdate(update Update, allowed map[int64]struct{}, receivedAt time.Time) (Post, SkipReason, bool) {
	msg := update.ChannelPost
	edited := false
	if msg == nil {
		msg, edited = update.EditedChannelPost, true
	}
	if msg == nil || msg.Chat.Type != "channel" {
		return Post{}, "", false
	}
	text := msg.Text
	if strings.TrimSpace(text) == "" {
		text = msg.Caption
	}
	post := Post{ChatID: msg.Chat.ID, MessageID: msg.MessageID, ChannelUsername: msg.Chat.Username,
		ChannelTitle: msg.Chat.Title, Text: text, TelegramDate: time.Unix(msg.Date, 0).UTC(), ReceivedAt: receivedAt.UTC(), Edited: edited}
	if edited && msg.EditDate > 0 {
		post.TelegramDate = time.Unix(msg.EditDate, 0).UTC()
	}
	if usernamePattern.MatchString(msg.Chat.Username) && msg.MessageID > 0 {
		post.OriginalURL = "https://t.me/" + msg.Chat.Username + "/" + strconv.FormatInt(msg.MessageID, 10)
	}
	if _, ok := allowed[msg.Chat.ID]; !ok {
		return post, SkipSourceNotAllowed, true
	}
	return post, "", true
}
