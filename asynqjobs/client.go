package asynqjobs

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

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
	Mux       *asynq.ServeMux
	Inspector *asynq.Inspector
	Queries   *sqlc.Queries
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
	mux.Handle(AsynqJobTypeScheduled, NewScheduledJobProcessor(botClient))
	mux.Handle(AsynqJobTypePeriodic, NewPeriodicJobProcessor(botClient))

	return &Client{
		Client:    client,
		Server:    server,
		Scheduler: scheduler,
		Mux:       mux,
		Inspector: inspector,
		Queries:   queries,
	}
}

func (c *Client) AddScheduledJob(payload ScheduledJobPayload, schedule time.Time, jobID *string) (string, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal [payload: %+v][jobID: %v]: %w", bytes, jobID, err)
	}

	var opts []asynq.Option
	if jobID != nil {
		opts = append(opts, asynq.TaskID(*jobID))
	}
	task := asynq.NewTask(AsynqJobTypeScheduled, bytes, opts...)

	info, err := c.Client.Enqueue(task, asynq.ProcessAt(schedule))
	if err != nil {
		return "", fmt.Errorf("failed to enqueue scheduled task [payload: %+v][jobID: %v][schedule: %v]: %w", payload,
			jobID, schedule.String(), err)
	}
	return info.ID, nil
}

func (c *Client) AddPeriodicJob(payload PeriodicJobPayload, schedule string, jobID *string) (string, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal [payload: %+v][jobID: %v]: %w", bytes, jobID, err)
	}

	var opts []asynq.Option
	if jobID != nil {
		opts = append(opts, asynq.TaskID(*jobID))
	}
	task := asynq.NewTask(AsynqJobTypePeriodic, bytes, opts...)

	entryID, err := c.Scheduler.Register(schedule, task)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue periodic task [payload: %+v][jobID: %v][schedule: %v]: %w", payload,
			jobID, schedule, err)
	}
	return entryID, nil
}

func (c *Client) CancelPeriodicJob(jobID string) error {
	if err := c.Scheduler.Unregister(jobID); err != nil {
		return fmt.Errorf("failed to unregister periodic job [jobID: %v]: %w", jobID, err)
	}
	return nil
}

func (c *Client) CancelScheduledJob(jobID string) error {
	if err := c.Inspector.DeleteTask("default", jobID); err != nil {
		return fmt.Errorf("failed to de-queue scheduled job [jobID: %v]: %w", jobID, err)
	}
	return nil
}
