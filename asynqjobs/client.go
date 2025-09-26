package asynqjobs

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	mux.Handle(AsynqJobTypeScheduled, NewScheduledJobProcessor(botClient, queries))
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
	if err := c.Inspector.DeleteTask("default", asynqJobID); err != nil {
		return fmt.Errorf("failed to de-queue scheduled job [asynqJobID: %v]: %w", asynqJobID, err)
	}
	return nil
}
