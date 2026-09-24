package telegramcollector

import (
	"context"
	"strings"
	"testing"

	"jobhub-ai/backend/internal/webscanner"
)

type fakeSender struct{ messages []string }

func (f *fakeSender) SendMessage(_ context.Context, _ int64, s string) error {
	f.messages = append(f.messages, s)
	return nil
}

type fakeScanner struct{ calls int }

func (f *fakeScanner) Scan(context.Context, string) (webscanner.Result, error) {
	f.calls++
	return webscanner.Result{ID: "ws_test", Domain: "example.kz", Status: "succeeded"}, nil
}
func TestCommandsAreAdminOnly(t *testing.T) {
	s := &fakeSender{}
	scanner := &fakeScanner{}
	p := NewCommandProcessor(map[int64]struct{}{42: {}}, scanner, s, nil)
	handled, err := p.Process(context.Background(), 1, 7, "/scan https://example.kz", "ru")
	if err != nil || !handled || scanner.calls != 0 || len(s.messages) != 1 || !strings.Contains(s.messages[0], "Доступ") {
		t.Fatalf("unauthorized command escaped: %v %v %#v", handled, err, s.messages)
	}
	_, _ = p.Process(context.Background(), 1, 42, "/scan https://example.kz", "kk")
	if scanner.calls != 1 || !strings.Contains(strings.ToLower(s.messages[1]), "dry-run") {
		t.Fatalf("admin scan failed: %#v", s.messages)
	}
}

func TestStartReturnsNumericIDWithoutAuthorization(t *testing.T) {
	s := &fakeSender{}
	p := NewCommandProcessor(map[int64]struct{}{}, &fakeScanner{}, s, nil)
	handled, err := p.Process(context.Background(), 9, 123456, "/start", "kk")
	if err != nil || !handled || len(s.messages) != 1 || !strings.Contains(s.messages[0], "123456") {
		t.Fatalf("start failed: %#v %v", s.messages, err)
	}
}
