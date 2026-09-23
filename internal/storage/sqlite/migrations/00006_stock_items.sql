-- +goose Up
-- +goose StatementBegin
-- name は食材マスタを持たない自由テキスト。recipe_ingredients.name と同時に
-- add-ingredient-master で FK 列へ移行する想定。
CREATE TABLE IF NOT EXISTS stock_items (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL CHECK (length(name) > 0),
    amount     REAL NOT NULL CHECK (amount > 0),
    unit       TEXT NOT NULL DEFAULT '',
    expires_at TEXT,
    added_at   TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_stock_items_name ON stock_items (name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS stock_items;
-- +goose StatementEnd
