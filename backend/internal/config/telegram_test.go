package config

import "testing"

func TestLoadTelegram(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "secret-value")
	t.Setenv("TELEGRAM_ALLOWED_CHANNEL_IDS", "-1001,-1002")
	t.Setenv("TELEGRAM_OFFSET_FILE", t.TempDir()+"/offset")
	cfg, err := LoadTelegram()
	if err != nil || len(cfg.Allowed) != 2 || cfg.Token != "secret-value" {
		t.Fatalf("unexpected config: %#v %v", cfg, err)
	}
}

func TestLoadTelegramRequiresTokenAndNumericChannels(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	if _, err := LoadTelegram(); err == nil {
		t.Fatal("expected missing token error")
	}
	t.Setenv("TELEGRAM_BOT_TOKEN", "secret")
	t.Setenv("TELEGRAM_ALLOWED_CHANNEL_IDS", "public-name")
	if _, err := LoadTelegram(); err == nil {
		t.Fatal("expected invalid allowlist error")
	}
}

func TestValidateTelegramDevelopmentDatabase(t *testing.T) {
	good := "postgresql://user:password@127.0.0.1:5432/jobhub_telegram_poc_test?sslmode=disable"
	if err := ValidateTelegramDevelopmentDatabase("development", good); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"postgresql://host/jobhub_telegram_poc_test", "postgresql://127.0.0.1:5432/production", good + "&search_path=public"} {
		if ValidateTelegramDevelopmentDatabase("development", raw) == nil {
			t.Fatalf("accepted unsafe database %q", raw)
		}
	}
	if ValidateTelegramDevelopmentDatabase("production", good) == nil {
		t.Fatal("accepted production environment")
	}
}
