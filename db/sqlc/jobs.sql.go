// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: jobs.sql

package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createJob = `-- name: CreateJob :one
INSERT INTO jobs (telegram_chat_id, is_recurring, message, schedule, name, river_job_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, telegram_chat_id, is_recurring, river_job_id, message, schedule, name, created_at, updated_at, deleted_at
`

type CreateJobParams struct {
	TelegramChatID int64
	IsRecurring    bool
	Message        string
	Schedule       string
	Name           string
	RiverJobID     pgtype.Int8
}

func (q *Queries) CreateJob(ctx context.Context, arg CreateJobParams) (Job, error) {
	row := q.db.QueryRow(ctx, createJob,
		arg.TelegramChatID,
		arg.IsRecurring,
		arg.Message,
		arg.Schedule,
		arg.Name,
		arg.RiverJobID,
	)
	var i Job
	err := row.Scan(
		&i.ID,
		&i.TelegramChatID,
		&i.IsRecurring,
		&i.RiverJobID,
		&i.Message,
		&i.Schedule,
		&i.Name,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
	)
	return i, err
}

const deleteJob = `-- name: DeleteJob :one
UPDATE jobs
SET deleted_at = NOW()
WHERE river_job_id = $1
RETURNING id, telegram_chat_id, is_recurring, river_job_id, message, schedule, name, created_at, updated_at, deleted_at
`

func (q *Queries) DeleteJob(ctx context.Context, riverJobID pgtype.Int8) (Job, error) {
	row := q.db.QueryRow(ctx, deleteJob, riverJobID)
	var i Job
	err := row.Scan(
		&i.ID,
		&i.TelegramChatID,
		&i.IsRecurring,
		&i.RiverJobID,
		&i.Message,
		&i.Schedule,
		&i.Name,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
	)
	return i, err
}
