-- name: GetChat :one
SELECT id, telegram_chat_id, context
FROM chats
WHERE telegram_chat_id = $1
AND deleted_at IS NULL;

-- name: CreateChat :one
INSERT INTO chats (telegram_chat_id)
VALUES ($1)
RETURNING *;

-- name: UpdateChatContext :one
UPDATE chats
SET context = $1
WHERE telegram_chat_id = $2
RETURNING *;
