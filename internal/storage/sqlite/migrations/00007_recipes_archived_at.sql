-- +goose Up
-- +goose StatementBegin
-- 論理削除。NULL = 未削除。献立の履歴を残すため行は消さない (add-recipe-deletion design.md D1)。
ALTER TABLE recipes ADD COLUMN archived_at TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE recipes DROP COLUMN archived_at;
-- +goose StatementEnd
