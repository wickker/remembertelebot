package asynqjobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"

	"remembertelebot/bot"
	"remembertelebot/db/sqlc"
)

type ScheduledJobProcessor struct {
	botClient *bot.Client
	queries   *sqlc.Queries
}

type ScheduledJobPayload struct {
	Message    string `json:"message"`
	ChatID     int64  `json:"chat_id"`
	AsynqJobID string `json:"asynq_job_id"`
}

func NewScheduledJobProcessor(botClient *bot.Client, queries *sqlc.Queries) *ScheduledJobProcessor {
	return &ScheduledJobProcessor{
		botClient: botClient,
		queries:   queries,
	}
}

func (p *ScheduledJobProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload ScheduledJobPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal scheduled job payload [payload: %+v]: %w", t.Payload(), err)
	}

	// send message
	if err := p.botClient.SendPlainMessage(payload.ChatID, payload.Message); err != nil {
		return fmt.Errorf("failed to send scheduled message [payload: %+v]: %w", payload, err)
	}

	// delete the job
	if _, err := p.queries.DeleteScheduledJobByAsynqJobID(context.Background(), pgtype.Text{Valid: true,
		String: payload.AsynqJobID}); err != nil {
		return fmt.Errorf("failed to delete job by asynqJobID [payload: %+v]: %w", payload, err)
	}
	return nil
}
