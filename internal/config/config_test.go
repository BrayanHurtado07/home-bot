package config

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	// Test default value
	val := getEnv("NON_EXISTENT_VAR_12345", "default_val")
	if val != "default_val" {
		t.Errorf("expected default_val, got %s", val)
	}

	// Test value from environment
	os.Setenv("TEST_ENV_VAR_abc", "hello")
	defer os.Unsetenv("TEST_ENV_VAR_abc")

	val = getEnv("TEST_ENV_VAR_abc", "default_val")
	if val != "hello" {
		t.Errorf("expected hello, got %s", val)
	}
}

func TestLoad(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "mock_token")
	os.Setenv("GROQ_API_KEY", "mock_key")
	os.Setenv("COCKROACH_DB_URL", "postgresql://user@localhost:5432/db")
	os.Setenv("USE_LONG_POLLING", "false")

	defer func() {
		os.Unsetenv("TELEGRAM_BOT_TOKEN")
		os.Unsetenv("GROQ_API_KEY")
		os.Unsetenv("COCKROACH_DB_URL")
		os.Unsetenv("USE_LONG_POLLING")
	}()

	cfg := Load()

	if cfg.TelegramBotToken != "mock_token" {
		t.Errorf("expected mock_token, got %s", cfg.TelegramBotToken)
	}
	if cfg.GroqAPIKey != "mock_key" {
		t.Errorf("expected mock_key, got %s", cfg.GroqAPIKey)
	}
	if cfg.CockroachDBURL != "postgresql://user@localhost:5432/db" {
		t.Errorf("expected postgresql://user@localhost:5432/db, got %s", cfg.CockroachDBURL)
	}
	if cfg.UseLongPolling != false {
		t.Errorf("expected false, got %t", cfg.UseLongPolling)
	}
}
