-- name: CreateJob :one
INSERT INTO jobs (telegram_chat_id, is_recurring, message, schedule, name, river_job_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;
