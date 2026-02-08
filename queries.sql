-- name: AddParticipant :exec
INSERT INTO participants (chat_id, user_id, first_name, username)
VALUES (?, ?, ?, ?)
ON CONFLICT (chat_id, user_id) DO UPDATE SET
    first_name = excluded.first_name,
    username = excluded.username;

-- name: RemoveParticipant :execresult
DELETE FROM participants WHERE chat_id = ? AND user_id = ?;

-- name: GetParticipants :many
SELECT user_id, first_name, username
FROM participants
WHERE chat_id = ?
ORDER BY joined_at;

-- name: GetTodayResult :one
SELECT chat_id, user_id, played_date
FROM results
WHERE chat_id = ? AND played_date = ?;

-- name: SaveResult :exec
INSERT INTO results (chat_id, user_id, played_date)
VALUES (?, ?, ?);

-- name: GetStats :many
SELECT p.user_id, p.first_name, p.username, COUNT(r.id) AS wins
FROM participants p
LEFT JOIN results r ON p.chat_id = r.chat_id AND p.user_id = r.user_id
WHERE p.chat_id = ?
GROUP BY p.user_id, p.first_name, p.username
ORDER BY wins DESC, p.first_name;

-- name: GetStatsByYear :many
SELECT p.user_id, p.first_name, p.username, COUNT(r.id) AS wins
FROM participants p
JOIN results r ON p.chat_id = r.chat_id AND p.user_id = r.user_id
WHERE p.chat_id = ? AND r.played_date >= ? AND r.played_date < ?
GROUP BY p.user_id, p.first_name, p.username
ORDER BY wins DESC, p.first_name;

-- name: GetParticipantByID :one
SELECT first_name, username
FROM participants
WHERE chat_id = ? AND user_id = ?;

-- name: DeleteTodayResult :execresult
DELETE FROM results WHERE chat_id = ? AND played_date = ?;

-- name: GetRandomMessageSetID :one
SELECT id FROM message_sets ORDER BY RANDOM() LIMIT 1;

-- name: GetSetMessages :many
SELECT body FROM set_messages
WHERE set_id = ? ORDER BY position;

-- name: GetAllTranslations :many
SELECT key, value FROM translations;
