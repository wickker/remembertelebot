// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: chats.sql

package sqlc

import (
	"context"
)

const listChats = `-- name: ListChats :many
SELECT id, telegram_chat_id, context, created_at, updated_at, deleted_at
FROM chats
`

func (q *Queries) ListChats(ctx context.Context) ([]Chat, error) {
	rows, err := q.db.Query(ctx, listChats)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Chat
	for rows.Next() {
		var i Chat
		if err := rows.Scan(
			&i.ID,
			&i.TelegramChatID,
			&i.Context,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.DeletedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
