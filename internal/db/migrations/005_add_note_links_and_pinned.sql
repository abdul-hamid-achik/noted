-- Note links for bidirectional linking (wikilinks)
CREATE TABLE IF NOT EXISTS note_links (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source_note_id INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  target_note_id INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  link_text TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(source_note_id, target_note_id, link_text)
);

CREATE INDEX IF NOT EXISTS idx_note_links_source ON note_links(source_note_id);
CREATE INDEX IF NOT EXISTS idx_note_links_target ON note_links(target_note_id);

-- Pin/star support
ALTER TABLE notes ADD COLUMN pinned BOOLEAN DEFAULT FALSE;
ALTER TABLE notes ADD COLUMN pinned_at DATETIME;
