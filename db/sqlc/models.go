// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0

package sqlc

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Chat struct {
	ID             int32
	TelegramChatID int32
	Context        []byte
	CreatedAt      pgtype.Timestamp
	UpdatedAt      pgtype.Timestamp
	DeletedAt      pgtype.Timestamp
}
