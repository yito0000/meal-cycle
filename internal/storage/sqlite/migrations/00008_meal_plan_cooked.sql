-- +goose Up
-- +goose StatementBegin
-- 作った印は (batch_id, recipe_id) の組ごとに 1 行 (add-mealplan-cooked-mark design.md D1/D2)。
-- meal_plans の主キーは (date, meal, position) で (batch_id, recipe_id) は一意でないため、外部キーは張らない。
CREATE TABLE meal_plan_cooked (
    batch_id  TEXT NOT NULL CHECK (length(batch_id) > 0),
    recipe_id TEXT NOT NULL,
    cooked_at TEXT NOT NULL,
    PRIMARY KEY (batch_id, recipe_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE meal_plan_cooked;
-- +goose StatementEnd
