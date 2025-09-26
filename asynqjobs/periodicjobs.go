package asynqjobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	"remembertelebot/bot"
)

type PeriodicJobProcessor struct {
	botClient *bot.Client
}

type PeriodicJobPayload struct {
	Message string `json:"message"`
	ChatID  int64  `json:"chat_id"`
}

func NewPeriodicJobProcessor(botClient *bot.Client) *PeriodicJobProcessor {
	return &PeriodicJobProcessor{
		botClient: botClient,
	}
}

func (p *PeriodicJobProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload PeriodicJobPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal periodic job payload [payload: %+v]: %w", t.Payload(), err)
	}

	log.Info().Msgf("Processing periodic job: %+v", payload)
	// TODO:
	// send message
	return nil
}
