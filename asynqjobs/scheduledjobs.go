package asynqjobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	"remembertelebot/bot"
)

type ScheduledJobProcessor struct {
	botClient *bot.Client
}

type ScheduledJobPayload struct {
	Message string `json:"message"`
	ChatID  int64  `json:"chat_id"`
}

func NewScheduledJobProcessor(botClient *bot.Client) *ScheduledJobProcessor {
	return &ScheduledJobProcessor{
		botClient: botClient,
	}
}

func (p *ScheduledJobProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload ScheduledJobPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal [payload: %+v]: %w", t.Payload(), err)
	}

	log.Info().Msgf("Processing scheduled job: %+v", payload)
	// TODO:
	return nil
}
