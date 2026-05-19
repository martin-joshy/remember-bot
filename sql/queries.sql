-- name: GetUserByLID :one
SELECT * FROM users
WHERE l_id = ? LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (l_id, phone_number, display_name)
VALUES (?, ?, ?)
RETURNING *;

-- name: CreateMessage :one
INSERT INTO messages (user_id, stanza_id, sent_at, type)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: CreateMessageAttachment :one
INSERT INTO message_attachments (message_id, body)
VALUES (?, ?)
RETURNING *;

-- name: ListMessagesByUserTag :many
SELECT m.* FROM messages m
JOIN message_tags mt ON mt.message_id = m.id
JOIN tags t ON t.id = mt.tag_id
WHERE t.name = ? AND t.user_id = ?
ORDER BY m.sent_at DESC;

-- name: GetTagsByMessageID :many
SELECT t.* FROM tags t
JOIN message_tags mt ON mt.tag_id = t.id
WHERE mt.message_id = ?;

-- name: CreateSentTaggedMessage :one
INSERT INTO sent_tagged_messages (stanza_id, original_message_id, user_id, sent_at)
VALUES (?, ?, ?, ?)
RETURNING * ;

-- name: GetSentTaggedMessageByStanzaID :one
SELECT * FROM sent_tagged_messages
WHERE stanza_id = ? LIMIT 1;

-- name: DeleteSentTaggedMessageByOriginalMessageID :exec
DELETE FROM sent_tagged_messages
WHERE original_message_id = ?;

-- name: GetMessageAttachmentByMessageID :many
SELECT * FROM message_attachments
WHERE message_id = ?;
