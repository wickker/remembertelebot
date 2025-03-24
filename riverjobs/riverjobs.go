package riverjobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"

	"remembertelebot/config"
)

type Client struct {
	Client *river.Client[pgx.Tx]
}

func NewClient(envCfg config.EnvConfig, pool *pgxpool.Pool) *Client {
	return &Client{
		Client: setupRiverClient(envCfg, pool),
	}
}

func (c *Client) AddScheduledJob(message string, schedule time.Time) (*int64, error) {
	job, err := c.Client.Insert(context.Background(), ScheduledJobArgs{
		Message: message,
	}, &river.InsertOpts{
		ScheduledAt: schedule,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add scheduled job [message: %s][schedule: %s]: %w", message,
			schedule.String(),
			err)
	}

	return &job.Job.ID, nil
}

func (c *Client) CancelScheduledJob(jobID int64) error {
	if _, err := c.Client.JobCancel(context.Background(), jobID); err != nil {
		return fmt.Errorf("failed to cancel scheduled job [jobID: %d]: %w", jobID, err)
	}
	return nil
}

func (c *Client) AddPeriodicJob(message string, cronTab string) (*int64, error) {
	schedule, err := cron.ParseStandard(cronTab)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cron tab [cronTab: %s]: %w", cronTab, err)
	}

	jobHandle := c.Client.PeriodicJobs().Add(river.NewPeriodicJob(
		schedule,
		func() (river.JobArgs, *river.InsertOpts) {
			return PeriodicJobArgs{
				Message: message,
			}, nil
		},
		nil,
	))

	jobHandleInt64 := int64(jobHandle)
	return &jobHandleInt64, nil
}

func (c *Client) CancelPeriodicJob(jobHandle int64) {
	jobHandleInt := int(jobHandle)
	c.Client.PeriodicJobs().Remove(rivertype.PeriodicJobHandle(jobHandleInt))
}

func setupRiverClient(envCfg config.EnvConfig, pool *pgxpool.Pool) *river.Client[pgx.Tx] {
	workers := river.NewWorkers()
	river.AddWorker(workers, &ScheduledJobWorker{})
	river.AddWorker(workers, &PeriodicJobWorker{})

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger: slog.Default(),
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		TestOnly: envCfg.IsDev(),
		Workers:  workers,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to initialize new River client.")
	}

	if err := riverClient.Start(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Unable to start River client.")
	}

	return riverClient
}
