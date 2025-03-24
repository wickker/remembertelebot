package riverjobs

import (
	"context"

	"github.com/riverqueue/river"
)

type ScheduledJobArgs struct {
	Message string `json:"message"`
}

func (ScheduledJobArgs) Kind() string { return "scheduled" }

type ScheduledJobWorker struct {
	river.WorkerDefaults[ScheduledJobArgs]
}

func (w *ScheduledJobWorker) Work(ctx context.Context, job *river.Job[ScheduledJobArgs]) error {
	//fmt.Printf("Message: %s\n", job.Args.Message)
	// TODO: Send message
	return nil
}
