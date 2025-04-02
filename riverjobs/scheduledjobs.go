package riverjobs

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"

	"remembertelebot/bot"
)

type ScheduledJobArgs struct {
	Message string `json:"message"`
	ChatID  int64  `json:"chat_id"`
}

func (ScheduledJobArgs) Kind() string { return "scheduled" }

type ScheduledJobWorker struct {
	river.WorkerDefaults[ScheduledJobArgs]
	botClient *bot.Client
}

func NewScheduledJobWorker(botClient *bot.Client) *ScheduledJobWorker {
	return &ScheduledJobWorker{
		botClient: botClient,
	}
}

func (w *ScheduledJobWorker) Work(ctx context.Context, job *river.Job[ScheduledJobArgs]) error {
	if err := w.botClient.SendPlainMessage(job.Args.ChatID, job.Args.Message); err != nil {
		return fmt.Errorf("failed to send scheduled message [jobArgs: %+v]: %w", job.Args, err)
	}
	return nil
}
