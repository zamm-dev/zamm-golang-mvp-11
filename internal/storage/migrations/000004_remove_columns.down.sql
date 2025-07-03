-- migrations/000004_remove_columns.down.sql
ALTER TABLE spec_nodes ADD COLUMN stable_id TEXT NOT NULL;
ALTER TABLE spec_nodes ADD COLUMN version INTEGER NOT NULL;
ALTER TABLE spec_nodes ADD COLUMN node_type TEXT NOT NULL DEFAULT 'spec';
