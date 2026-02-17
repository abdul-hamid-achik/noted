-- FTS5 full-text search virtual table (external content from notes)
CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
  title,
  content,
  content='notes',
  content_rowid='id'
);

-- Populate FTS index from existing data
INSERT INTO notes_fts(rowid, title, content) SELECT id, title, content FROM notes;

-- Triggers to keep FTS index in sync with notes table

-- After INSERT: add new row to FTS
CREATE TRIGGER IF NOT EXISTS notes_fts_insert AFTER INSERT ON notes BEGIN
  INSERT INTO notes_fts(rowid, title, content) VALUES (new.id, new.title, new.content);
END;

-- After UPDATE: remove old row and add updated row
CREATE TRIGGER IF NOT EXISTS notes_fts_update AFTER UPDATE ON notes BEGIN
  INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES ('delete', old.id, old.title, old.content);
  INSERT INTO notes_fts(rowid, title, content) VALUES (new.id, new.title, new.content);
END;

-- After DELETE: remove row from FTS
CREATE TRIGGER IF NOT EXISTS notes_fts_delete AFTER DELETE ON notes BEGIN
  INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES ('delete', old.id, old.title, old.content);
END;
