package messages

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
	log.Info().Msgf("Received message from %s: [message: %s][chatID: %v][sticker: %+v]", message.From.UserName,
		message.Text, message.Chat.ID, message.Sticker)

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
		// process 1st input of /newjob
		h.processJobName(message, chatContextMap)
		return
	}

	if _, exists := chatContextMap["name"]; exists && len(chatContextMap) == 1 {
		// process 2nd input of /newjob
		h.processJobMessage(message, chatContextMap)
		return
	}

	if _, exists := chatContextMap["is_recurring"]; exists && len(chatContextMap) == 3 {
		// process 4th input of /newjob
		h.processJobSchedule(message, chatContextMap)
		return
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
	name, err := validateJobName(message.Text)
	if err != nil {
		h.sendErrorMessage(err, message)
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

	if err := h.botClient.SendPlainMessage(message.Chat.ID, "Please input the message to be scheduled."); err != nil {
		log.Err(err).Msgf("Unable to send request for job message [user: %s].", message.From.UserName)
		return
	}
}

func (h *Handler) processJobMessage(message *tgbotapi.Message, contextMap map[string]string) {
	text, err := validateJobMessage(message.Text)
	if err != nil {
		h.sendErrorMessage(err, message)
		return
	}

	contextMap["message"] = text
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
	if err := h.botClient.SendHtmlMessage(message.Chat.ID, "Select message schedule type.", buttons); err != nil {
		log.Err(err).Msgf("Unable to send html message [telegramChatID: %v].", message.Chat.ID)
		h.sendErrorMessage(err, message)
		return
	}
}

func (h *Handler) processJobSchedule(message *tgbotapi.Message, contextMap map[string]string) {
	isRecurring := contextMap["is_recurring"]
	var (
		schedule string
		err      error
	)

	if isRecurring == "true" {
		schedule, err = validateCronTab(message.Text)
		if err != nil {
			// TODO:
			h.sendErrorMessage(err, message)
			return
		}
	} else {
		ts, err := validateScheduleTimestamp(message.Text)
		if err != nil {
			h.sendErrorMessage(err, message)
			return
		}
		schedule = ts.Format(time.DateTime)
	}

	contextMap["schedule"] = schedule
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

	button := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Confirm", callbackqueries.ConfirmJobQueryData),
		))

	confirmationMsg := generateConfirmationMessage(contextMap)
	if err := h.botClient.SendHtmlMessage(message.Chat.ID, confirmationMsg, button); err != nil {
		log.Err(err).Msgf("Unable to send html message [telegramChatID: %v].", message.Chat.ID)
		h.sendErrorMessage(err, message)
		return
	}
}
