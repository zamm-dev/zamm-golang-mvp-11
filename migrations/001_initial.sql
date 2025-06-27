-- migrations/001_initial.sql
CREATE TABLE spec_nodes (
    id TEXT PRIMARY KEY,
    stable_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    node_type TEXT NOT NULL DEFAULT 'spec',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(stable_id, version)
);

CREATE INDEX idx_spec_nodes_stable_id ON spec_nodes(stable_id);
CREATE INDEX idx_spec_nodes_created_at ON spec_nodes(created_at);

CREATE TABLE spec_commit_links (
    id TEXT PRIMARY KEY,
    spec_id TEXT NOT NULL,
    commit_id TEXT NOT NULL,
    repo_path TEXT NOT NULL,
    link_type TEXT NOT NULL DEFAULT 'implements',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (spec_id) REFERENCES spec_nodes(id) ON DELETE CASCADE,
    UNIQUE(spec_id, commit_id, repo_path)
);

CREATE INDEX idx_links_spec_id ON spec_commit_links(spec_id);
CREATE INDEX idx_links_commit_id ON spec_commit_links(commit_id);
CREATE INDEX idx_links_repo_path ON spec_commit_links(repo_path);