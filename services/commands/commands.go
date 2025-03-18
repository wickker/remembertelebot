package commands

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"

	"remembertelebot/bot"
)

const (
	StartCommand = "start"
)

type Handler struct {
	botClient *bot.Client
}

func NewHandler(botClient *bot.Client) *Handler {
	return &Handler{
		botClient: botClient,
	}
}

func (h *Handler) ProcessCommand(update tgbotapi.Update) {
	log.Info().Msgf("Received command from %s: [command: %s][chatID: %v]", update.Message.From.UserName,
		update.Message.Text, update.Message.Chat.ID)

	switch update.Message.Command() {
	case StartCommand:
		h.processStart(update.Message)
	default:
		h.processDefault(update.Message)
	}
}

func (h *Handler) processStart(message *tgbotapi.Message) {
	// TODO:
	if err := h.botClient.SendPlainMessage(message.Chat.ID, "Received start."); err != nil {
		log.Err(err).Msgf("Unable to respond to start command [User: %s].", message.From.UserName)
		return
	}
}

func (h *Handler) processDefault(message *tgbotapi.Message) {
	// TODO:
	if err := h.botClient.SendPlainMessage(message.Chat.ID, "Received unknown command."); err != nil {
		log.Err(err).Msgf("Unable to respond to unknown command [User: %s].", message.From.UserName)
		return
	}
}
