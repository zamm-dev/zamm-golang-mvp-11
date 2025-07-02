-- Rollback migration for project metadata table

DROP TRIGGER IF EXISTS project_metadata_updated_at;
DROP TABLE IF EXISTS project_metadata;
