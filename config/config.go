package config

import "strings"

type EnvConfig struct {
	Env              string `env:"ENV" envDefault:"dev"`
	DatabaseURL      string `env:"DATABASE_URL"`
	TelegramBotToken string `env:"TELEGRAM_BOT_TOKEN"`
	BaseURL          string `env:"BASE_URL"`
	DeepSeekAPIKey   string `env:"DEEP_SEEK_API_KEY"`
}

func (c EnvConfig) IsDev() bool {
	return strings.EqualFold(c.Env, "dev")
}
