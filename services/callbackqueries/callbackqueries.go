package callbackqueries

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"remembertelebot/bot"
	"remembertelebot/db/sqlc"
	"remembertelebot/riverjobs"
)

const (
	ScheduledQueryData  = "scheduled"
	PeriodicQueryData   = "periodic"
	ConfirmJobQueryData = "confirm-job"
)

type Handler struct {
	botClient   *bot.Client
	queries     *sqlc.Queries
	riverClient *riverjobs.Client
	pool        *pgxpool.Pool
}

func NewHandler(botClient *bot.Client, queries *sqlc.Queries, riverClient *riverjobs.Client, pool *pgxpool.Pool) *Handler {
	return &Handler{
		botClient:   botClient,
		queries:     queries,
		riverClient: riverClient,
		pool:        pool,
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
	case ConfirmJobQueryData:
		h.processConfirmJob(query)
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

func (h *Handler) processConfirmJob(query *tgbotapi.CallbackQuery) {
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

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		log.Err(err).Msgf("Unable to begin tx [chat: %+v].", chat)
		h.sendErrorMessage(err, query)
		return
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	isRecurring, err := strconv.ParseBool(chatContextMap["is_recurring"])
	if err != nil {
		log.Err(err).Msgf("Unable to parse boolean [isRecurring: %v][chat: %+v].", chatContextMap["is_recurring"],
			chat)
		h.sendErrorMessage(err, query)
		return
	}

	var riverJobID *int64
	if isRecurring {
		riverJobID, err = h.riverClient.AddPeriodicJob(chatContextMap["message"], chat.TelegramChatID,
			chatContextMap["schedule"])
		if err != nil {
			log.Err(err).Msgf("Unable to add periodic job to river client [chat: %+v].",
				chat)
			h.sendErrorMessage(err, query)
			return
		}

	} else {
		schedule, err := time.Parse(time.DateTime, chatContextMap["schedule"])
		if err != nil {
			log.Err(err).Msgf("Unable to parse once-off schedule to time [schedule: %v][chat: %+v].",
				chatContextMap["schedule"], chat)
			h.sendErrorMessage(err, query)
			return
		}
		riverJobID, err = h.riverClient.AddScheduledJobTx(tx, chatContextMap["message"], chat.TelegramChatID, schedule)
		if err != nil {
			log.Err(err).Msgf("Unable to add scheduled job to river client [chat: %+v].",
				chat)
			h.sendErrorMessage(err, query)
			return
		}
	}

	if riverJobID == nil {
		err := errors.New("river job ID is nil")
		log.Err(err).Msgf("Unable to obtain valid river job ID [chat: %+v].",
			chat)
		h.sendErrorMessage(err, query)
		return
	}

	qtx := h.queries.WithTx(tx)
	if _, err := qtx.CreateJob(ctx, sqlc.CreateJobParams{
		TelegramChatID: query.Message.Chat.ID,
		IsRecurring:    isRecurring,
		Message:        chatContextMap["message"],
		Schedule:       chatContextMap["schedule"],
		Name:           chatContextMap["name"],
		RiverJobID:     pgtype.Int8{Valid: true, Int64: *riverJobID},
	}); err != nil {
		log.Err(err).Msgf("Unable to add new job to db [chat: %+v][riverJobID: %v].",
			chat, *riverJobID)
		h.sendErrorMessage(err, query)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		log.Err(err).Msgf("Unable to commit tx [chat: %+v][riverJobID: %v].",
			chat, *riverJobID)
		h.sendErrorMessage(err, query)
		return
	}

	// show loader
	_ = h.botClient.SendCallbackConfig(query.ID, "")

	// edit the previous html message with confirmation button
	text := fmt.Sprintf("Successfully scheduled job %s", chatContextMap["name"])
	if err := h.botClient.SendEditMessage(query.Message.Chat.ID, query.Message.MessageID, text); err != nil {
		log.Err(err).Msgf("Unable to edit html markup to send success message [user: %s].",
			query.From.UserName)
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

	// show loader
	_ = h.botClient.SendCallbackConfig(query.ID, "")

	// edit the previous html message with buttons
	text := "Please input the UTC date and time in the format YYYY-MM-DD HH:MM:SS that the once-off message should be" +
		" sent."
	if isRecurring == "true" {
		text = "Please input the UTC cron expression (i.e. * * * * * *) that the recurring message should be sent."
	}
	if err := h.botClient.SendEditMessage(query.Message.Chat.ID, query.Message.MessageID, text); err != nil {
		log.Err(err).Msgf("Unable to edit html markup to send request for schedule [user: %s].",
			query.From.UserName)
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
