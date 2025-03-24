package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"

	"remembertelebot/bot"
	"remembertelebot/db/sqlc"
)

const (
	StartCommand  = "start"
	NewJobCommand = "newjob"
)

type Handler struct {
	botClient *bot.Client
	queries   *sqlc.Queries
}

func NewHandler(botClient *bot.Client, queries *sqlc.Queries) *Handler {
	return &Handler{
		botClient: botClient,
		queries:   queries,
	}
}

func (h *Handler) ProcessCommand(update tgbotapi.Update) {
	log.Info().Msgf("Received command from %s: [command: %s][chatID: %v]", update.Message.From.UserName,
		update.Message.Text, update.Message.Chat.ID)

	switch update.Message.Command() {
	case StartCommand:
		h.processStart(update.Message)
	case NewJobCommand:
		h.processNewJob(update.Message)
	default:
		h.processDefault(update.Message)
	}
}

func (h *Handler) processStart(message *tgbotapi.Message) {
	// TODO: Change this to return list of commands
	if err := h.botClient.SendPlainMessage(message.Chat.ID, "Received start."); err != nil {
		log.Err(err).Msgf("Unable to respond to /start command [user: %s].", message.From.UserName)
		return
	}
}

func (h *Handler) processDefault(message *tgbotapi.Message) {
	if err := h.botClient.SendPlainMessage(message.Chat.ID, "Received unknown command."); err != nil {
		log.Err(err).Msgf("Unable to respond to unknown command [user: %s].", message.From.UserName)
		return
	}
}

func (h *Handler) processNewJob(message *tgbotapi.Message) {
	ctx := context.Background()
	_, err := h.queries.GetChat(ctx, message.Chat.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Err(err).Msgf("Unable to get chat [telegramChatID: %v].", message.Chat.ID)
		h.sendErrorMessage(err, message)
		return
	}

	if errors.Is(err, sql.ErrNoRows) {
		if _, err := h.queries.CreateChat(ctx, message.Chat.ID); err != nil {
			log.Err(err).Msgf("Unable to create chat [telegramChatID: %v].", message.Chat.ID)
			h.sendErrorMessage(err, message)
			return
		}
	} else {
		if _, err := h.queries.UpdateChatContext(ctx, sqlc.UpdateChatContextParams{
			TelegramChatID: message.Chat.ID,
			Context:        []byte("{}"),
		}); err != nil {
			log.Err(err).Msgf("Unable to update empty chat context [telegramChatID: %v].", message.Chat.ID)
			h.sendErrorMessage(err, message)
			return
		}
	}

	if err := h.botClient.SendPlainMessage(message.Chat.ID, "Please enter a name for your job."); err != nil {
		log.Err(err).Msgf("Unable to respond to /newjob command [user: %s].", message.From.UserName)
		return
	}
}

func (h *Handler) sendErrorMessage(err error, message *tgbotapi.Message) {
	if err := h.botClient.SendPlainMessage(message.Chat.ID, fmt.Sprintf("An error occurred processing the command: %v",
		err.Error())); err != nil {
		log.Warn().Err(err).Msgf("Unable to publish error message [user: %s][message: %v].", message.From.UserName,
			err.Error())
	}
}
