package riverjobs

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"

	"remembertelebot/bot"
)

type PeriodicJobArgs struct {
	Message string `json:"message"`
	ChatID  int64  `json:"chat_id"`
}

func (PeriodicJobArgs) Kind() string { return "periodic" }

type PeriodicJobWorker struct {
	river.WorkerDefaults[PeriodicJobArgs]
	botClient *bot.Client
}

func NewPeriodicJobWorker(botClient *bot.Client) *PeriodicJobWorker {
	return &PeriodicJobWorker{
		botClient: botClient,
	}
}

func (w *PeriodicJobWorker) Work(ctx context.Context, job *river.Job[PeriodicJobArgs]) error {
	if err := w.botClient.SendPlainMessage(job.Args.ChatID, job.Args.Message); err != nil {
		return fmt.Errorf("failed to send periodic message [jobArgs: %+v]: %w", job.Args, err)
	}
	return nil
}
