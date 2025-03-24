package messages

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"

	"remembertelebot/bot"
)

type Handler struct {
	botClient *bot.Client
}

func NewHandler(botClient *bot.Client) *Handler {
	return &Handler{
		botClient: botClient,
	}
}

func (h *Handler) ProcessMessage(message *tgbotapi.Message) {
	log.Info().Msgf("Received message from %s: [message: %s][chatID: %v]", message.From.UserName,
		message.Text, message.Chat.ID)

	if err := h.botClient.SendPlainMessage(message.Chat.ID, "Received message."); err != nil {
		log.Err(err).Msgf("Unable to respond to message [user: %s].", message.From.UserName)
		return
	}
}
