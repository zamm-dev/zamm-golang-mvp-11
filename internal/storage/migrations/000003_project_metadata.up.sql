-- Migration to add project metadata table for tracking project-level information
-- This stores well-defined project metadata with proper foreign key constraints

CREATE TABLE IF NOT EXISTS project_metadata (
    id INTEGER PRIMARY KEY CHECK (id = 1), -- Ensures only one row exists
    root_spec_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (root_spec_id) REFERENCES spec_nodes(id) ON DELETE SET NULL
);

-- Create trigger to update the updated_at timestamp
CREATE TRIGGER IF NOT EXISTS project_metadata_updated_at
    AFTER UPDATE ON project_metadata
BEGIN
    UPDATE project_metadata SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
