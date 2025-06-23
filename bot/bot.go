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
	bot.Debug = envCfg.IsDev()

	webhook, err := tgbotapi.NewWebhook(fmt.Sprintf("%s/webhooks", envCfg.BaseURL))
	if err != nil {
		return nil, err
	}
	if _, err := bot.Request(webhook); err != nil {
		return nil, err
	}

	return &Client{
		bot: bot,
	}, nil
}

// Old implementation for reference:
//func (c *Client) CreateBotChannel() tgbotapi.UpdatesChannel {
//	cfg := tgbotapi.NewUpdate(0)
//	cfg.Timeout = 60
//	return c.Bot.GetUpdatesChan(cfg)
//}

func (c *Client) SendPlainMessage(chatID int64, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	if _, err := c.bot.Send(msg); err != nil {
		return fmt.Errorf("bot failed to send plain message [messageConfig: %+v]: %w", msg, err)
	}
	return nil
}

func (c *Client) SendCallbackConfig(queryID, text string) error {
	callbackCfg := tgbotapi.NewCallback(queryID, text)
	if _, err := c.bot.Send(callbackCfg); err != nil {
		return fmt.Errorf("bot failed to send callback config [callbackConfig: %+v]: %w", callbackCfg, err)
	}
	return nil
}

func (c *Client) SendHtmlMessage(chatID int64, text string, markup interface{}) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = markup

	if _, err := c.bot.Send(msg); err != nil {
		return fmt.Errorf("bot failed to send html message [messageConfig: %+v]: %w", msg, err)
	}
	return nil
}

func (c *Client) SendEditMessage(chatID int64, messageID int, text string) error {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)

	if _, err := c.bot.Send(msg); err != nil {
		return fmt.Errorf("bot failed to send edit message [messageConfig: %+v]: %w", msg, err)
	}
	return nil
}
