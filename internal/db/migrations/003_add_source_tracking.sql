-- Migration 003: Add source tracking columns

ALTER TABLE notes ADD COLUMN source TEXT;
ALTER TABLE notes ADD COLUMN source_ref TEXT;
