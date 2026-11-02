-- +goose Up
-- +goose StatementBegin
-- 0 は未入力。NULL にすると bindings の生成形式で nullable の付き方が揃わないため (add-mealplan-suggest-by-time design.md D2)。
ALTER TABLE recipes ADD COLUMN cook_minutes INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE recipes DROP COLUMN cook_minutes;
-- +goose StatementEnd
