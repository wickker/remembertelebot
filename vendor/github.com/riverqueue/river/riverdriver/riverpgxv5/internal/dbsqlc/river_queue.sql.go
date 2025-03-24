// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: river_queue.sql

package dbsqlc

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

const queueCreateOrSetUpdatedAt = `-- name: QueueCreateOrSetUpdatedAt :one
INSERT INTO river_queue(
    created_at,
    metadata,
    name,
    paused_at,
    updated_at
) VALUES (
    now(),
    coalesce($1::jsonb, '{}'::jsonb),
    $2::text,
    coalesce($3::timestamptz, NULL),
    coalesce($4::timestamptz, now())
) ON CONFLICT (name) DO UPDATE
SET
    updated_at = coalesce($4::timestamptz, now())
RETURNING name, created_at, metadata, paused_at, updated_at
`

type QueueCreateOrSetUpdatedAtParams struct {
	Metadata  []byte
	Name      string
	PausedAt  *time.Time
	UpdatedAt *time.Time
}

func (q *Queries) QueueCreateOrSetUpdatedAt(ctx context.Context, db DBTX, arg *QueueCreateOrSetUpdatedAtParams) (*RiverQueue, error) {
	row := db.QueryRow(ctx, queueCreateOrSetUpdatedAt,
		arg.Metadata,
		arg.Name,
		arg.PausedAt,
		arg.UpdatedAt,
	)
	var i RiverQueue
	err := row.Scan(
		&i.Name,
		&i.CreatedAt,
		&i.Metadata,
		&i.PausedAt,
		&i.UpdatedAt,
	)
	return &i, err
}

const queueDeleteExpired = `-- name: QueueDeleteExpired :many
DELETE FROM river_queue
WHERE name IN (
    SELECT name
    FROM river_queue
    WHERE updated_at < $1::timestamptz
    ORDER BY name ASC
    LIMIT $2::bigint
)
RETURNING name, created_at, metadata, paused_at, updated_at
`

type QueueDeleteExpiredParams struct {
	UpdatedAtHorizon time.Time
	Max              int64
}

func (q *Queries) QueueDeleteExpired(ctx context.Context, db DBTX, arg *QueueDeleteExpiredParams) ([]*RiverQueue, error) {
	rows, err := db.Query(ctx, queueDeleteExpired, arg.UpdatedAtHorizon, arg.Max)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*RiverQueue
	for rows.Next() {
		var i RiverQueue
		if err := rows.Scan(
			&i.Name,
			&i.CreatedAt,
			&i.Metadata,
			&i.PausedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const queueGet = `-- name: QueueGet :one
SELECT name, created_at, metadata, paused_at, updated_at
FROM river_queue
WHERE name = $1::text
`

func (q *Queries) QueueGet(ctx context.Context, db DBTX, name string) (*RiverQueue, error) {
	row := db.QueryRow(ctx, queueGet, name)
	var i RiverQueue
	err := row.Scan(
		&i.Name,
		&i.CreatedAt,
		&i.Metadata,
		&i.PausedAt,
		&i.UpdatedAt,
	)
	return &i, err
}

const queueList = `-- name: QueueList :many
SELECT name, created_at, metadata, paused_at, updated_at
FROM river_queue
ORDER BY name ASC
LIMIT $1::integer
`

func (q *Queries) QueueList(ctx context.Context, db DBTX, limitCount int32) ([]*RiverQueue, error) {
	rows, err := db.Query(ctx, queueList, limitCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*RiverQueue
	for rows.Next() {
		var i RiverQueue
		if err := rows.Scan(
			&i.Name,
			&i.CreatedAt,
			&i.Metadata,
			&i.PausedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const queuePause = `-- name: QueuePause :execresult
WITH queue_to_update AS (
    SELECT name, paused_at
    FROM river_queue
    WHERE CASE WHEN $1::text = '*' THEN true ELSE name = $1 END
    FOR UPDATE
),
updated_queue AS (
    UPDATE river_queue
    SET
        paused_at = now(),
        updated_at = now()
    FROM queue_to_update
    WHERE river_queue.name = queue_to_update.name
        AND river_queue.paused_at IS NULL
    RETURNING river_queue.name, river_queue.created_at, river_queue.metadata, river_queue.paused_at, river_queue.updated_at
)
SELECT name, created_at, metadata, paused_at, updated_at
FROM river_queue
WHERE name = $1
    AND name NOT IN (SELECT name FROM updated_queue)
UNION
SELECT name, created_at, metadata, paused_at, updated_at
FROM updated_queue
`

func (q *Queries) QueuePause(ctx context.Context, db DBTX, name string) (pgconn.CommandTag, error) {
	return db.Exec(ctx, queuePause, name)
}

const queueResume = `-- name: QueueResume :execresult
WITH queue_to_update AS (
    SELECT name
    FROM river_queue
    WHERE CASE WHEN $1::text = '*' THEN true ELSE river_queue.name = $1::text END
    FOR UPDATE
),
updated_queue AS (
    UPDATE river_queue
    SET
        paused_at = NULL,
        updated_at = now()
    FROM queue_to_update
    WHERE river_queue.name = queue_to_update.name
    RETURNING river_queue.name, river_queue.created_at, river_queue.metadata, river_queue.paused_at, river_queue.updated_at
)
SELECT name, created_at, metadata, paused_at, updated_at
FROM river_queue
WHERE name = $1
    AND name NOT IN (SELECT name FROM updated_queue)
UNION
SELECT name, created_at, metadata, paused_at, updated_at
FROM updated_queue
`

func (q *Queries) QueueResume(ctx context.Context, db DBTX, name string) (pgconn.CommandTag, error) {
	return db.Exec(ctx, queueResume, name)
}
