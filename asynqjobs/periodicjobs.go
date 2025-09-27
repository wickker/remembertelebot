package asynqjobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"

	"remembertelebot/bot"
)

type PeriodicJobProcessor struct {
	botClient *bot.Client
}

type PeriodicJobPayload struct {
	Message    string `json:"message"`
	ChatID     int64  `json:"chat_id"`
	AsynqJobID string `json:"asynq_job_id"`
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

	if err := p.botClient.SendPlainMessage(payload.ChatID, payload.Message); err != nil {
		return fmt.Errorf("failed to send periodic message [payload: %+v]: %w", payload, err)
	}
	return nil
}
