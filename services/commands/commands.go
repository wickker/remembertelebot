package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"

	"remembertelebot/bot"
	"remembertelebot/db/sqlc"
	"remembertelebot/riverjobs"
	"remembertelebot/services/messages"
)

const (
	StartCommand     = "start"
	NewJobCommand    = "newjob"
	ListJobsCommand  = "listjobs"
	CancelJobCommand = "canceljob"
)

type Handler struct {
	botClient   *bot.Client
	queries     *sqlc.Queries
	riverClient *riverjobs.Client
}

func NewHandler(botClient *bot.Client, queries *sqlc.Queries, riverClient *riverjobs.Client) *Handler {
	return &Handler{
		botClient:   botClient,
		queries:     queries,
		riverClient: riverClient,
	}
}

func (h *Handler) ProcessCommand(update tgbotapi.Update) {
	log.Info().Msgf("Received command from %s: [command: %s][chatID: %v]", update.Message.From.UserName,
		update.Message.Command(), update.Message.Chat.ID)

	command := update.Message.Command()
	switch {
	case command == StartCommand:
		h.processStart(update.Message)
	case command == NewJobCommand:
		h.processNewJob(update.Message)
	case command == ListJobsCommand:
		h.processListJobs(update.Message)
	case command == CancelJobCommand:
		h.processCancelJob(update.Message)
	default:
		h.processDefault(update.Message)
	}
}

func (h *Handler) processStart(message *tgbotapi.Message) {
	startText := "Welcome to RememberOrDismember! 🐟\n\n" +
		"Are you tired of having a goldfish memory? Well, you're in luck! I'm here to help you remember things, " +
		"or else... *sharpens virtual knife* 🗡️\n\n" +
		"Whether it's a one-time task or something you need to be reminded of regularly, I'll make sure you don't forget. " +
		"Because if you do... well, let's just say I have a very creative way of helping people remember things.\n\n" +
		"There are two types of reminders you can set:\n" +
		"1. Once-off reminders - Perfect for one-time tasks or events (or else...)\n" +
		"2. Recurring reminders - Great for regular tasks that need to be done periodically (or you'll be dismembered periodically)\n\n" +
		"Available commands:\n" +
		"/start - Show this help menu\n" +
		"/newjob - Create a new reminder job\n" +
		"/listjobs - List all your active reminder jobs\n" +
		"/canceljob-<jobID> - Cancel a specific job (e.g. /canceljob-123)\n\n" +
		"To create a new job, use /newjob and follow the prompts to set up your reminder. " +
		"Remember, I'm watching... always watching... 👀"

	if err := h.botClient.SendPlainMessage(message.Chat.ID, startText); err != nil {
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

func (h *Handler) processCancelJob(message *tgbotapi.Message) {
	command := message.Text
	jobIDStr := strings.TrimPrefix(command, "/canceljob-")
	if jobIDStr == "" {
		log.Error().Msgf("Invalid job ID [command: %s].", command)
		h.sendErrorMessage(errors.New("please provide a valid job ID"), message)
		return
	}

	var jobID int32
	if _, err := fmt.Sscanf(jobIDStr, "%d", &jobID); err != nil {
		log.Err(err).Msgf("Invalid job ID format [command: %s].", command)
		h.sendErrorMessage(errors.New("please provide a valid numeric job ID"), message)
		return
	}

	job, err := h.queries.GetJobByID(context.Background(), jobID)
	if err != nil {
		log.Err(err).Msgf("Unable to get job [jobID: %v].", jobID)
		h.sendErrorMessage(err, message)
		return
	}

	if job.TelegramChatID != message.Chat.ID {
		log.Err(err).Msgf("Unauthorized job cancellation [telegramChatID: %v][job: %+v].", message.Chat.ID, job)
		h.sendErrorMessage(errors.New("you can only cancel your own jobs"), message)
		return
	}

	if job.IsRecurring {
		h.riverClient.CancelPeriodicJob(job.RiverJobID.Int64)
	} else {
		if err := h.riverClient.CancelScheduledJob(job.RiverJobID.Int64); err != nil {
			log.Err(err).Msgf("Unable to cancel scheduled job on river [riverJobID: %v].", job.RiverJobID.Int64)
			h.sendErrorMessage(err, message)
			return
		}
	}

	if _, err := h.queries.DeleteJobByID(context.Background(), job.ID); err != nil {
		log.Err(err).Msgf("Unable to delete job in db [jobID: %v].", jobID)
		h.sendErrorMessage(err, message)
		return
	}

	if err := h.botClient.SendPlainMessage(message.Chat.ID, fmt.Sprintf("Successfully cancelled job: %s", job.Name)); err != nil {
		log.Err(err).Msgf("Unable to send success message for job cancellation [user: %s][jobID: %v].", message.From.UserName, jobID)
		return
	}
}

func (h *Handler) processListJobs(message *tgbotapi.Message) {
	jobs, err := h.queries.GetActiveJobsByTelegramChatID(context.Background(), message.Chat.ID)
	if err != nil {
		log.Err(err).Msgf("Unable to get active jobs [telegramChatID: %v].", message.Chat.ID)
		h.sendErrorMessage(err, message)
		return
	}

	var jobsText string
	if len(jobs) == 0 {
		jobsText = "You have no jobs yet. Input /newjob to create a new job."
	} else {
		for _, job := range jobs {
			scheduleText := fmt.Sprintf("Once-off, at UTC %s", job.Schedule)
			if job.IsRecurring {
				scheduleText = fmt.Sprintf("Recurring at UTC %s (%s)", job.Schedule, messages.GetCronDescriptor(job.Schedule))
			}

			jobText := fmt.Sprintf("Job ID: %v\nJob name: %s\nMessage: %s\nSchedule: %s\n\n", job.ID, job.Name, job.Message, scheduleText)
			jobsText += jobText
		}

		jobsText += "To cancel a job, " +
			"input the command /canceljob-<jobID> where jobID is the ID of the job you want to cancel.\n\nFor example, " +
			"if jobID is 123, you would input /canceljob-123."
	}

	if err := h.botClient.SendPlainMessage(message.Chat.ID, jobsText); err != nil {
		log.Err(err).Msgf("Unable to respond to /listjobs command [user: %s].", message.From.UserName)
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
