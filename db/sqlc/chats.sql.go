// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: chats.sql

package sqlc

import (
	"context"
)

const createChat = `-- name: CreateChat :one
INSERT INTO chats (telegram_chat_id)
VALUES ($1)
RETURNING id, telegram_chat_id, context, created_at, updated_at, deleted_at
`

func (q *Queries) CreateChat(ctx context.Context, telegramChatID int32) (Chat, error) {
	row := q.db.QueryRow(ctx, createChat, telegramChatID)
	var i Chat
	err := row.Scan(
		&i.ID,
		&i.TelegramChatID,
		&i.Context,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
	)
	return i, err
}

const getChat = `-- name: GetChat :one
SELECT id, telegram_chat_id, context
FROM chats
WHERE telegram_chat_id = $1
AND deleted_at IS NULL
`

type GetChatRow struct {
	ID             int32
	TelegramChatID int32
	Context        []byte
}

func (q *Queries) GetChat(ctx context.Context, telegramChatID int32) (GetChatRow, error) {
	row := q.db.QueryRow(ctx, getChat, telegramChatID)
	var i GetChatRow
	err := row.Scan(&i.ID, &i.TelegramChatID, &i.Context)
	return i, err
}

const updateChatContext = `-- name: UpdateChatContext :one
UPDATE chats
SET context = $1
WHERE telegram_chat_id = $2
RETURNING id, telegram_chat_id, context, created_at, updated_at, deleted_at
`

type UpdateChatContextParams struct {
	Context        []byte
	TelegramChatID int32
}

func (q *Queries) UpdateChatContext(ctx context.Context, arg UpdateChatContextParams) (Chat, error) {
	row := q.db.QueryRow(ctx, updateChatContext, arg.Context, arg.TelegramChatID)
	var i Chat
	err := row.Scan(
		&i.ID,
		&i.TelegramChatID,
		&i.Context,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
	)
	return i, err
}
