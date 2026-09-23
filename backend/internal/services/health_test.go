package services

import (
	"context"
	"errors"
	"testing"
	"time"
)

type pinger struct{ check func(context.Context) error }

func (p pinger) Ping(ctx context.Context) error { return p.check(ctx) }
func TestHealthTimeoutAndError(t *testing.T) {
	sentinel := errors.New("database down")
	service := NewHealthService(pinger{func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 2*time.Second {
			t.Fatal("missing bounded timeout")
		}
		return sentinel
	}})
	if err := service.Check(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("lost database error: %v", err)
	}
}
func TestHealthPreservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := NewHealthService(pinger{func(ctx context.Context) error { return ctx.Err() }})
	if !errors.Is(service.Check(ctx), context.Canceled) {
		t.Fatal("cancellation lost")
	}
}
