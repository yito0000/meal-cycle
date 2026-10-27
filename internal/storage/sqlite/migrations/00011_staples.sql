-- +goose Up
-- +goose StatementBegin
-- 常備品は量を持たないので stock_items とは別の表にする (add-fridge-staples design.md D1)。
-- 名前の重複は表記をそろえて比べるためサービスで検査し、UNIQUE は付けない (D3)。
CREATE TABLE IF NOT EXISTS staples (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL CHECK (length(name) > 0),
    state      TEXT NOT NULL CHECK (state IN ('enough', 'low', 'out')),
    added_at   TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_staples_name ON staples (name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS staples;
-- +goose StatementEnd
