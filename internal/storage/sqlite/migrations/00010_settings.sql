-- +goose Up
-- +goose StatementBegin
-- 既定値から変えた項目だけを持つ。既定値の見直しを、値を触っていない利用者にも届けるため (add-settings-screen design.md D2)。
CREATE TABLE settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE settings;
-- +goose StatementEnd
