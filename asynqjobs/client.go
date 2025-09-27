package asynqjobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	"remembertelebot/bot"
	"remembertelebot/config"
	"remembertelebot/db/sqlc"
)

const (
	AsynqJobTypePeriodic  = "periodic"
	AsynqJobTypeScheduled = "scheduled"
)

type Client struct {
	Client    *asynq.Client
	Scheduler *asynq.Scheduler
	Server    *asynq.Server
	Mux       *asynq.ServeMux // multiplex
	inspector *asynq.Inspector
	queries   *sqlc.Queries
}

func NewClient(envCfg config.EnvConfig, botClient *bot.Client, queries *sqlc.Queries) *Client {
	redisOpt := asynq.RedisClientOpt{
		Addr:     envCfg.RedisAddress,
		Username: envCfg.RedisUsername,
		Password: envCfg.RedisPassword,
	}
	client := asynq.NewClient(redisOpt)
	scheduler := asynq.NewScheduler(redisOpt, nil)
	inspector := asynq.NewInspector(redisOpt)
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{Concurrency: 10},
	)
	mux := asynq.NewServeMux()
	mux.Handle(AsynqJobTypeScheduled, NewScheduledJobProcessor(botClient, queries))
	mux.Handle(AsynqJobTypePeriodic, NewPeriodicJobProcessor(botClient))

	c := &Client{
		Client:    client,
		Server:    server,
		Scheduler: scheduler,
		Mux:       mux,
		inspector: inspector,
		queries:   queries,
	}
	if err := c.addJobsOnStartUpIfNotExists(); err != nil {
		log.Err(err).Msgf("Unable to add active jobs on start up.")
	}

	return c
}

func (c *Client) AddScheduledJob(payload ScheduledJobPayload, schedule time.Time, jobID *string) (string, error) {
	var taskID string
	if jobID == nil {
		taskID = uuid.New().String()
	} else {
		taskID = *jobID
	}
	payload.AsynqJobID = taskID

	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal [payload: %+v][jobID: %v]: %w", bytes, jobID, err)
	}
	task := asynq.NewTask(AsynqJobTypeScheduled, bytes, asynq.TaskID(taskID))

	info, err := c.Client.Enqueue(task, asynq.ProcessAt(schedule))
	if err != nil {
		return "", fmt.Errorf("failed to enqueue scheduled task [payload: %+v][jobID: %v][schedule: %v]: %w", payload,
			jobID, schedule.String(), err)
	}
	return info.ID, nil
}

func (c *Client) AddPeriodicJob(payload PeriodicJobPayload, schedule string, jobID *string) (string, error) {
	var taskID string
	if jobID == nil {
		taskID = uuid.New().String()
	} else {
		taskID = *jobID
	}
	payload.AsynqJobID = taskID

	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal [payload: %+v][jobID: %v]: %w", bytes, jobID, err)
	}
	task := asynq.NewTask(AsynqJobTypePeriodic, bytes, asynq.TaskID(taskID))

	entryID, err := c.Scheduler.Register(schedule, task)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue periodic task [payload: %+v][jobID: %v][schedule: %v]: %w", payload,
			jobID, schedule, err)
	}
	return entryID, nil
}

func (c *Client) CancelPeriodicJob(asynqJobID string) error {
	if err := c.Scheduler.Unregister(asynqJobID); err != nil {
		return fmt.Errorf("failed to unregister periodic job [asynqJobID: %v]: %w", asynqJobID, err)
	}
	return nil
}

func (c *Client) CancelScheduledJob(asynqJobID string) error {
	if err := c.inspector.DeleteTask("default", asynqJobID); err != nil {
		return fmt.Errorf("failed to de-queue scheduled job [asynqJobID: %v]: %w", asynqJobID, err)
	}
	return nil
}

func (c *Client) addJobsOnStartUpIfNotExists() error {
	jobs, err := c.queries.GetActiveJobs(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get active jobs: %w", err)
	}

	for _, job := range jobs {
		// recurring jobs
		if job.IsRecurring {
			payload := PeriodicJobPayload{
				Message: job.Message,
				ChatID:  job.TelegramChatID,
			}
			if _, err := c.AddPeriodicJob(payload, job.Schedule, &job.AsynqJobID.String); err != nil {
				if errors.Is(err, asynq.ErrDuplicateTask) {
					log.Warn().Msgf("Periodic job is already scheduled [asyncJobID: %v].", job.AsynqJobID.String)
					continue
				}
				log.Err(err).Msgf("Unable to schedule periodic job [asyncJobID: %v].", job.AsynqJobID.String)
			}
			continue
		}

		// once-off jobs
		payload := ScheduledJobPayload{
			Message: job.Message,
			ChatID:  job.TelegramChatID,
		}
		schedule, err := time.Parse(time.DateTime, job.Schedule)
		if err != nil {
			log.Err(err).Msgf("Unable to parse once-off schedule to time [schedule: %v][jobID: %v].",
				job.Schedule, job.ID)
			continue
		}
		if _, err := c.AddScheduledJob(payload, schedule, &job.AsynqJobID.String); err != nil {
			if errors.Is(err, asynq.ErrDuplicateTask) {
				log.Warn().Msgf("Scheduled job is already queued [asyncJobID: %v].", job.AsynqJobID.String)
				continue
			}
			log.Err(err).Msgf("Unable to enqueue scheduled job [asyncJobID: %v].", job.AsynqJobID.String)
		}
	}

	return nil
}
