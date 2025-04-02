-- name: CreateJob :one
INSERT INTO jobs (telegram_chat_id, is_recurring, message, schedule, name, river_job_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetJobByID :one
SELECT id, telegram_chat_id, is_recurring, message, schedule, name, river_job_id
FROM jobs
WHERE id = $1
AND deleted_at IS NULL;

-- name: DeleteJobByID :one
UPDATE jobs
SET deleted_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteScheduledJobByRiverJobID :one
UPDATE jobs
SET deleted_at = NOW()
WHERE river_job_id = $1
AND is_recurring = false
RETURNING *;

-- name: GetActiveRecurringJobs :many
SELECT id, telegram_chat_id, is_recurring, message, schedule, name, river_job_id
FROM jobs
WHERE is_recurring = true
AND deleted_at IS NULL;

-- name: GetActiveJobsByTelegramChatID :many
SELECT id, telegram_chat_id, is_recurring, message, schedule, name, river_job_id
FROM jobs
WHERE telegram_chat_id = $1
AND deleted_at IS NULL;

-- name: UpdateRiverJobID :one
UPDATE jobs
SET river_job_id = $1
WHERE id = $2
RETURNING *;
