-- Migration 002: Add expires_at column for TTL support

ALTER TABLE notes ADD COLUMN expires_at DATETIME;

CREATE INDEX IF NOT EXISTS idx_notes_expires_at ON notes(expires_at) WHERE expires_at IS NOT NULL;
