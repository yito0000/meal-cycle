-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS schema_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
INSERT OR IGNORE INTO schema_meta (key, value)
VALUES ('scaffold_installed_at', datetime('now'));
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS schema_meta;
