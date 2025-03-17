package config

import "strings"

type EnvConfig struct {
	Env              string `env:"ENV" envDefault:"dev"`
	DatabaseURL      string `env:"DATABASE_URL"`
	TelegramBotToken string `env:"TELEGRAM_BOT_TOKEN"`
}

func (c EnvConfig) IsDev() bool {
	return strings.EqualFold(c.Env, "dev")
}
