package riverjobs

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"
)

type PeriodicJobArgs struct {
	Message string `json:"message"`
}

func (PeriodicJobArgs) Kind() string { return "periodic" }

type PeriodicJobWorker struct {
	river.WorkerDefaults[PeriodicJobArgs]
}

func (w *PeriodicJobWorker) Work(ctx context.Context, job *river.Job[PeriodicJobArgs]) error {
	fmt.Printf("Message: %s\n", job.Args.Message)
	// TODO: Send message
	return nil
}
