package telegramcollector

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"jobhub-ai/backend/internal/ingestion"
	"jobhub-ai/backend/internal/jobs"
	telegramprovider "jobhub-ai/backend/internal/providers/telegram"
)

type SourceRegistrar interface {
	RegisterDevelopmentSource(context.Context, string, string, string) error
}

type Processor struct {
	allowed   map[int64]struct{}
	registrar SourceRegistrar
	store     jobs.Store
	logger    *slog.Logger
}

type Outcome struct {
	Handled    bool
	Accepted   bool
	ChatID     int64
	MessageID  int64
	SkipReason telegramprovider.SkipReason
	ExternalID string
	Report     ingestion.Report
}

func NewProcessor(allowed map[int64]struct{}, registrar SourceRegistrar, store jobs.Store, logger *slog.Logger) *Processor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Processor{allowed: allowed, registrar: registrar, store: store, logger: logger}
}

func (p *Processor) Process(ctx context.Context, update telegramprovider.Update) (Outcome, error) {
	post, reason, handled := telegramprovider.MapUpdate(update, p.allowed, time.Now())
	out := Outcome{Handled: handled, SkipReason: reason, ChatID: post.ChatID, MessageID: post.MessageID}
	if !handled || reason != "" {
		return out, nil
	}
	item, reason := telegramprovider.Normalize(post)
	if reason != "" {
		out.SkipReason = reason
		return out, nil
	}
	if p.registrar == nil || p.store == nil {
		return out, errors.New("Telegram ingestion store is unavailable")
	}
	displayName := strings.TrimSpace(post.ChannelTitle)
	if displayName == "" {
		displayName = item.Source
	}
	if err := p.registrar.RegisterDevelopmentSource(ctx, item.Source, "telegram", displayName); err != nil {
		return out, err
	}
	report, err := ingestion.NewService(p.store, nil, p.logger, 0).Run(ctx, telegramprovider.Collection{Item: item}, ingestion.RunOptions{NoAutomaticExpiry: true})
	if err != nil {
		return out, err
	}
	out.Accepted, out.ExternalID, out.Report = true, item.ExternalID, report
	return out, nil
}
