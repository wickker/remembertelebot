package messages

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cohesion-org/deepseek-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jsuar/go-cron-descriptor/pkg/crondescriptor"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"

	"remembertelebot/deepseekai"
)

func validateJobName(text string) (string, error) {
	name := strings.TrimSpace(text)
	if len(name) < 1 {
		return "", errors.New("job name is too short")
	}
	if len(name) > 191 {
		return "", errors.New("job name is too long")
	}
	return name, nil
}

func validateJobMessage(text string) (string, error) {
	msg := strings.TrimSpace(text)
	if len(text) < 1 {
		return "", errors.New("job name is too short")
	}
	return msg, nil
}

func validateScheduleTimestamp(text string) (time.Time, error) {
	text = strings.TrimSpace(text)
	now := time.Now()

	timestamp, err := time.Parse(time.DateTime, text)
	if err != nil {
		return now, err
	}

	if !timestamp.After(now) {
		return now, errors.New("timestamp must be in the future")
	}

	return timestamp, nil
}

func validateCronTab(text string) (string, error) {
	text = strings.TrimSpace(text)
	if _, err := cron.ParseStandard(text); err != nil {
		return "", err
	}
	return text, nil
}

func GetCronDescriptor(cronTab string) string {
	cd, _ := crondescriptor.NewCronDescriptor(cronTab)
	if cd != nil {
		description, _ := cd.GetDescription(crondescriptor.Full)
		return *description
	}
	return ""
}

func generateConfirmationMessage(contextMap map[string]string) string {
	name := contextMap["name"]
	isRecurring := contextMap["is_recurring"]
	message := contextMap["message"]
	schedule := contextMap["schedule"]

	scheduleText := fmt.Sprintf("Once-off, at UTC %s", schedule)
	if isRecurring == "true" {
		scheduleText = fmt.Sprintf("Recurring at UTC <b>%s</b> (%s)", schedule, GetCronDescriptor(schedule))
	}

	return fmt.Sprintf("Please confirm the following job details:\n\n<b>Job name:</b> %s\n<b>Message to send:</b> %s\n<b"+
		">Schedule"+
		":</b> %s"+
		"", name, message, scheduleText)
}

func (h *Handler) useAI(message *tgbotapi.Message) string {
	cacheKey := fmt.Sprintf("%d", message.Chat.ID)
	value, err := h.cache.Get(cacheKey)
	if err != nil {
		log.Warn().Err(err).Msgf("Unable to get cache key [cacheKey: %s].", cacheKey)
	}
	newMessage := deepseek.ChatCompletionMessage{
		Role:    deepseek.ChatMessageRoleUser,
		Content: message.Text,
	}

	// formulate messages array (depending on cache hit)
	messages := []deepseek.ChatCompletionMessage{{
		Role:    deepseek.ChatMessageRoleSystem,
		Content: deepseekai.Prompt,
	},
		newMessage,
	}
	if len(value) > 0 {
		messages = append(value, newMessage)
	}

	// get AI response to user
	aiResponse, err := h.deepSeekClient.Converse(messages)
	if err != nil {
		h.sendErrorMessage(err, message)
		return ""
	}

	// parse AI response
	if strings.Contains(aiResponse.Content, "final cron is ") {
		cronTab := strings.ReplaceAll(aiResponse.Content, "final cron is ", "")
		schedule, err := validateCronTab(cronTab)
		if err == nil {
			// clear cache
			return schedule
		}
	}

	// send AI response to user
	if err := h.botClient.SendPlainMessage(message.Chat.ID, aiResponse.Content); err != nil {
		h.sendErrorMessage(err, message)
	}

	// cache messages
	messages = append(messages, deepseek.ChatCompletionMessage{
		Role:    aiResponse.Role,
		Content: aiResponse.Content,
	})
	if err := h.cache.Set(cacheKey, messages); err != nil {
		log.Warn().Err(err).Msg("Unable to set cache.")
	}

	return ""
}
