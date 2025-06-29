-- migrations/002_spec_hierarchy.sql
-- Add support for hierarchical links between specifications (DAG structure)

CREATE TABLE spec_spec_links (
    id TEXT PRIMARY KEY,
    from_spec_id TEXT NOT NULL,
    to_spec_id TEXT NOT NULL,
    link_type TEXT NOT NULL DEFAULT 'child',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_spec_id) REFERENCES spec_nodes(id) ON DELETE CASCADE,
    FOREIGN KEY (to_spec_id) REFERENCES spec_nodes(id) ON DELETE CASCADE,
    UNIQUE(from_spec_id, to_spec_id),
    CHECK(from_spec_id != to_spec_id) -- Prevent self-linking
);

CREATE INDEX idx_spec_spec_links_from ON spec_spec_links(from_spec_id);
CREATE INDEX idx_spec_spec_links_to ON spec_spec_links(to_spec_id);
CREATE INDEX idx_spec_spec_links_type ON spec_spec_links(link_type);
