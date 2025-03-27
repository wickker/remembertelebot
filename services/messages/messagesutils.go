package messages

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jsuar/go-cron-descriptor/pkg/crondescriptor"
	"github.com/robfig/cron/v3"
)

const TimestampFormat = "2006-01-02 15:04:05"

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

	timestamp, err := time.Parse(TimestampFormat, text)
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

func getCronDescriptor(cronTab string) string {
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
		scheduleText = fmt.Sprintf("Recurring at UTC %s (%s)", schedule, getCronDescriptor(schedule))
	}

	return fmt.Sprintf("Please confirm the following job details:\n\nJob name: %s\nMessage to send: %s\nSchedule: %s"+
		"", name, message, scheduleText)
}
