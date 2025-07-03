-- migrations/000004_remove_columns.up.sql

-- SQLite does not support dropping columns or constraints directly
-- https://stackoverflow.com/a/42013422

DROP INDEX idx_spec_nodes_stable_id;
PRAGMA foreign_keys=off;

CREATE TABLE spec_nodes_new (
    id TEXT PRIMARY KEY,
    title TEXT,
    content TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

INSERT INTO spec_nodes_new (id, title, content, created_at, updated_at)
    SELECT id, title, content, created_at, updated_at FROM spec_nodes;

DROP TABLE spec_nodes;

ALTER TABLE spec_nodes_new RENAME TO spec_nodes;
PRAGMA foreign_keys=on;
