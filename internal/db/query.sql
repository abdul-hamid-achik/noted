-- name: CreateNote :one
INSERT INTO notes (title, content)
VALUES (?, ?)
RETURNING *;

-- name: GetNote :one
SELECT * FROM notes
WHERE id = ?;

-- name: ListNotes :many
SELECT * FROM notes
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateNote :one
UPDATE notes
SET title = ?, content = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteNote :exec
DELETE FROM notes
WHERE id = ?;

-- name: SearchNotesByTitle :many
SELECT * FROM notes
WHERE title LIKE ?
ORDER BY created_at DESC;

-- name: MarkEmbeddingSynced :exec
UPDATE notes
SET embedding_synced = TRUE
WHERE id = ?;

-- name: GetUnsynced :many
SELECT * FROM notes
WHERE embedding_synced = FALSE;

-- Tags --

-- name: CreateTag :one
INSERT INTO tags (name)
VALUES (?)
ON CONFLICT (name) DO UPDATE SET name = name
RETURNING *;

-- name: GetTagByName :one
SELECT * FROM tags
WHERE name = ?;

-- name: ListTags :many
SELECT * FROM tags
ORDER BY name;

-- name: DeleteTag :exec
DELETE FROM tags
WHERE id = ?;

-- Note Tags --

-- name: AddTagToNote :exec
INSERT INTO note_tags (note_id, tag_id)
VALUES (?, ?)
ON CONFLICT DO NOTHING;

-- name: RemoveTagFromNote :exec
DELETE FROM note_tags
WHERE note_id = ? AND tag_id = ?;

-- name: GetTagsForNote :many
SELECT t.* FROM tags t
INNER JOIN note_tags nt ON t.id = nt.tag_id
WHERE nt.note_id = ?
ORDER BY t.name;

-- name: GetNotesForTag :many
SELECT n.* FROM notes n
INNER JOIN note_tags nt ON n.id = nt.note_id
WHERE nt.tag_id = ?
ORDER BY n.created_at DESC;

-- name: GetNotesByTagName :many
SELECT n.* FROM notes n
INNER JOIN note_tags nt ON n.id = nt.note_id
INNER JOIN tags t ON nt.tag_id = t.id
WHERE t.name = ?
ORDER BY n.created_at DESC;

-- name: RemoveAllTagsFromNote :exec
DELETE FROM note_tags WHERE note_id = ?;

-- name: GetTagsWithCount :many
SELECT t.id, t.name, COUNT(nt.note_id) as note_count
FROM tags t
LEFT JOIN note_tags nt ON t.id = nt.tag_id
GROUP BY t.id
ORDER BY t.name;

-- name: DeleteUnusedTags :execrows
DELETE FROM tags
WHERE id NOT IN (SELECT DISTINCT tag_id FROM note_tags);

-- name: SearchNotesContent :many
SELECT * FROM notes
WHERE content LIKE ? OR title LIKE ?
ORDER BY updated_at DESC
LIMIT ?;

-- name: GetAllNotes :many
SELECT * FROM notes ORDER BY created_at DESC;
