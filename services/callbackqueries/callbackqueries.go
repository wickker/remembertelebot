package callbackqueries

import (
	"context"
	"encoding/json"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"

	"remembertelebot/bot"
	"remembertelebot/db/sqlc"
)

const (
	ScheduledQueryData = "scheduled"
	PeriodicQueryData  = "periodic"
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

func (h *Handler) ProcessCallbackQuery(query *tgbotapi.CallbackQuery) {
	log.Info().Msgf("Received callback query from %s: [queryData: %s][chatID: %v]", query.From.UserName,
		query.Data, query.Message.Chat.ID)

	switch query.Data {
	case ScheduledQueryData:
		h.processScheduled(query)
	case PeriodicQueryData:
		h.processPeriodic(query)
	default:
		h.processDefault(query)
	}
}

func (h *Handler) processDefault(query *tgbotapi.CallbackQuery) {
	if err := h.botClient.SendPlainMessage(query.Message.Chat.ID, "Received unknown query data."); err != nil {
		log.Err(err).Msgf("Unable to respond to unknown query data [user: %s].", query.From.UserName)
		return
	}
}

func (h *Handler) processJobType(query *tgbotapi.CallbackQuery, isRecurring string) {
	ctx := context.Background()

	chat, err := h.queries.GetChat(ctx, query.Message.Chat.ID)
	if err != nil {
		log.Err(err).Msgf("Unable to get chat [telegramChatID: %v].", query.Message.Chat.ID)
		h.sendErrorMessage(err, query)
		return
	}

	var chatContextMap map[string]string
	if err := json.Unmarshal(chat.Context, &chatContextMap); err != nil {
		log.Err(err).Msgf("Unable to unmarshal chat context [chat: %+v].", chat)
		h.sendErrorMessage(err, query)
		return
	}

	chatContextMap["is_recurring"] = isRecurring
	contextMapBytes, err := json.Marshal(chatContextMap)
	if err != nil {
		log.Err(err).Msgf("Unable to marshal chat context [contextMap: %+v].", chatContextMap)
		h.sendErrorMessage(err, query)
		return
	}

	if _, err := h.queries.UpdateChatContext(context.Background(), sqlc.UpdateChatContextParams{
		TelegramChatID: query.Message.Chat.ID,
		Context:        contextMapBytes,
	}); err != nil {
		log.Err(err).Msgf("Unable to update chat context [telegramChatID: %v][context: %+v].", query.Message.Chat.ID,
			chatContextMap)
		h.sendErrorMessage(err, query)
		return
	}

	msg := "Please input the UTC date and time that the once-off message should be sent."
	if isRecurring == "true" {
		msg = "Please input the UTC cron expression that the recurring message should be sent."
	}
	if err := h.botClient.SendPlainMessage(query.Message.Chat.ID, msg); err != nil {
		log.Err(err).Msgf("Unable to send schedule message [user: %s].", query.From.UserName)
		return
	}
}

func (h *Handler) processPeriodic(query *tgbotapi.CallbackQuery) {
	h.processJobType(query, "true")
}

func (h *Handler) processScheduled(query *tgbotapi.CallbackQuery) {
	h.processJobType(query, "false")
}

func (h *Handler) sendErrorMessage(err error, query *tgbotapi.CallbackQuery) {
	if err := h.botClient.SendPlainMessage(query.Message.Chat.ID,
		fmt.Sprintf("An error occurred processing the callback query: %v",
			err.Error())); err != nil {
		log.Warn().Err(err).Msgf("Unable to publish error message [user: %s][message: %v].",
			query.From.UserName,
			err.Error())
	}
}
