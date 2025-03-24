package messages

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"

	"remembertelebot/bot"
	"remembertelebot/db/sqlc"
	"remembertelebot/services/callbackqueries"
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

func (h *Handler) ProcessMessage(message *tgbotapi.Message) {
	log.Info().Msgf("Received message from %s: [message: %s][chatID: %v]", message.From.UserName,
		message.Text, message.Chat.ID)

	ctx := context.Background()
	chat, err := h.queries.GetChat(ctx, message.Chat.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Err(err).Msgf("Unable to get chat [telegramChatID: %v].", message.Chat.ID)
		h.sendErrorMessage(err, message)
		return
	}

	if errors.Is(err, sql.ErrNoRows) {
		h.processDefault(message)
		return
	}

	var chatContextMap map[string]string
	if err := json.Unmarshal(chat.Context, &chatContextMap); err != nil {
		log.Err(err).Msgf("Unable to unmarshal chat context [chat: %+v].", chat)
		h.sendErrorMessage(err, message)
		return
	}

	if len(chatContextMap) == 0 {
		h.processJobName(message, chatContextMap)
		return
	}

	if name, exists := chatContextMap["name"]; exists && len(chatContextMap) == 1 {
		fmt.Println(name)
		// TODO: Process 2nd message
	}

	if isRecurring, exists := chatContextMap["is_recurring"]; exists && len(chatContextMap) == 2 {
		fmt.Println(isRecurring)
		// TODO: Process 2nd message
	}

	h.processDefault(message)
}

func (h *Handler) sendErrorMessage(err error, message *tgbotapi.Message) {
	if err := h.botClient.SendPlainMessage(message.Chat.ID, fmt.Sprintf("An error occurred processing the message: %v",
		err.Error())); err != nil {
		log.Warn().Err(err).Msgf("Unable to publish error message [user: %s][message: %v].", message.From.UserName,
			err.Error())
	}
}

func (h *Handler) processDefault(message *tgbotapi.Message) {
	if err := h.botClient.SendPlainMessage(message.Chat.ID, "Unable to trace message context."+
		"\n\nDid you mean to enter a command? Please input /start to view the list of available commands."+
		""); err != nil {
		log.Err(err).Msgf("Unable to respond to unknown message context [user: %s].", message.From.UserName)
		return
	}
}

func (h *Handler) processJobName(message *tgbotapi.Message, contextMap map[string]string) {
	name := strings.TrimSpace(message.Text)
	if len(name) < 1 {
		h.sendErrorMessage(errors.New("job name is too short"), message)
		return
	}
	if len(name) > 191 {
		h.sendErrorMessage(errors.New("job name is too long"), message)
		return
	}

	contextMap["name"] = name
	contextMapBytes, err := json.Marshal(contextMap)
	if err != nil {
		log.Err(err).Msgf("Unable to marshal chat context [contextMap: %+v].", contextMap)
		h.sendErrorMessage(err, message)
		return
	}

	if _, err := h.queries.UpdateChatContext(context.Background(), sqlc.UpdateChatContextParams{
		TelegramChatID: message.Chat.ID,
		Context:        contextMapBytes,
	}); err != nil {
		log.Err(err).Msgf("Unable to update chat context [telegramChatID: %v][context: %+v].", message.Chat.ID, contextMap)
		h.sendErrorMessage(err, message)
		return
	}

	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Once-off", callbackqueries.ScheduledQueryData),
		),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Recurring", callbackqueries.PeriodicQueryData)),
	)
	if err := h.botClient.SendHtmlMessage(message.Chat.ID, "Select job type.", buttons); err != nil {
		log.Err(err).Msgf("Unable to send html message [telegramChatID: %v].", message.Chat.ID)
		h.sendErrorMessage(err, message)
		return
	}
}
