package bot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"remembertelebot/config"
)

type Client struct {
	bot *tgbotapi.BotAPI
}

func NewClient(envCfg config.EnvConfig) (*Client, error) {
	bot, err := tgbotapi.NewBotAPI(envCfg.TelegramBotToken)
	if err != nil {
		return nil, err
	}

	return &Client{
		bot: bot,
	}, nil
}

func (c *Client) CreateBotChannel() tgbotapi.UpdatesChannel {
	cfg := tgbotapi.NewUpdate(0)
	cfg.Timeout = 60
	return c.bot.GetUpdatesChan(cfg)
}

func (c *Client) SendPlainMessage(chatId int64, message string) error {
	msg := tgbotapi.NewMessage(chatId, message)
	if _, err := c.bot.Send(msg); err != nil {
		return fmt.Errorf("bot failed to send plain message [messageConfig: %+v]: %w", msg, err)
	}
	return nil
}
