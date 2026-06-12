package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramBotToken string
	GroqAPIKey       string
	CockroachDBURL   string
	UseLongPolling   bool
}

func Load() *Config {
	// Load .env file if it exists, otherwise just read env variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	useLongPolling := true
	if val, ok := os.LookupEnv("USE_LONG_POLLING"); ok {
		if parsed, err := strconv.ParseBool(val); err == nil {
			useLongPolling = parsed
		}
	}

	return &Config{
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		GroqAPIKey:       getEnv("GROQ_API_KEY", ""),
		CockroachDBURL:   getEnv("COCKROACH_DB_URL", "postgresql://root@localhost:26257/defaultdb?sslmode=disable"),
		UseLongPolling:   useLongPolling,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
