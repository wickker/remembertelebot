package riverjobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"

	"remembertelebot/bot"
	"remembertelebot/config"
	"remembertelebot/db/sqlc"
)

type Client struct {
	Client                 *river.Client[pgx.Tx]
	CancelCompletedChannel func()
	queries                *sqlc.Queries
}

func NewClient(envCfg config.EnvConfig, pool *pgxpool.Pool, botClient *bot.Client, queries *sqlc.Queries) *Client {
	client, completedChannel, cancelCompletedChannel := setupRiverClient(envCfg, pool, botClient)

	riverClient := &Client{
		Client:                 client,
		CancelCompletedChannel: cancelCompletedChannel,
		queries:                queries,
	}

	go riverClient.processJobCompletedEvent(completedChannel)

	return riverClient
}

func (c *Client) AddScheduledJobTx(tx pgx.Tx, message string, chatID int64, schedule time.Time) (*int64, error) {
	job, err := c.Client.InsertTx(context.Background(), tx, ScheduledJobArgs{
		Message: message,
		ChatID:  chatID,
	}, &river.InsertOpts{
		ScheduledAt: schedule,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add scheduled job tx [message: %s][schedule: %s]: %w", message,
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

func (c *Client) AddPeriodicJob(message string, chatID int64, cronTab string) (*int64, error) {
	schedule, err := cron.ParseStandard(cronTab)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cron tab [cronTab: %s]: %w", cronTab, err)
	}

	jobHandle := c.Client.PeriodicJobs().Add(river.NewPeriodicJob(
		schedule,
		func() (river.JobArgs, *river.InsertOpts) {
			return PeriodicJobArgs{
				Message: message,
				ChatID:  chatID,
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

func (c *Client) processJobCompletedEvent(subscribeChan <-chan *river.Event) {
	log.Info().Msg("Subscribed to river job completion event.")

	for {
		select {
		case event := <-subscribeChan:
			if event == nil {
				log.Info().Msg("River job completion event channel is closed.")
				return
			}

			log.Info().Msgf("Received river job completed event [riverJobID: %v][Kind: %v]", event.Job.ID,
				event.Job.Kind)

			if event.Job.Kind == "scheduled" {
				if _, err := c.queries.DeleteJob(context.Background(), pgtype.Int8{Valid: true,
					Int64: event.Job.ID}); err != nil {
					log.Err(err).Msgf("Unable to delete scheduled job [riverJobID: %v].", event.Job.ID)
				}
			}
		}
	}
}

func setupRiverClient(envCfg config.EnvConfig, pool *pgxpool.Pool, botClient *bot.Client) (*river.Client[pgx.Tx], <-chan *river.Event, func()) {
	workers := river.NewWorkers()
	river.AddWorker(workers, NewScheduledJobWorker(botClient))
	river.AddWorker(workers, NewPeriodicJobWorker(botClient))

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

	completedEventChannel, cancelCompletedEventChannel := riverClient.Subscribe(river.EventKindJobCompleted)

	if err := riverClient.Start(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Unable to start River client.")
	}

	return riverClient, completedEventChannel, cancelCompletedEventChannel
}
