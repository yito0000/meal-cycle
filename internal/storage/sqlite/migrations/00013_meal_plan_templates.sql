-- +goose Up
-- +goose StatementBegin
-- 名前の重複はサービスで検査し、UNIQUE は付けない。エラーの種類をサービスで選ぶため (add-mealplan-template design.md D1)。
CREATE TABLE IF NOT EXISTS meal_plan_templates (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL CHECK (length(name) > 0),
    created_at TEXT NOT NULL
);

-- 日付ではなく週の最初の日からのずれで持ち、適用は「対象の週の最初の日 + ずれ」で済ませる。
-- recipes は論理削除で行が残るので、外部キーに ON DELETE は付けない。
CREATE TABLE IF NOT EXISTS meal_plan_template_items (
    template_id TEXT    NOT NULL REFERENCES meal_plan_templates(id) ON DELETE CASCADE,
    day_offset  INTEGER NOT NULL CHECK (day_offset BETWEEN 0 AND 6),
    meal        TEXT    NOT NULL CHECK (meal IN ('breakfast', 'lunch', 'dinner')),
    position    INTEGER NOT NULL,
    recipe_id   TEXT    NOT NULL REFERENCES recipes(id),
    group_no    INTEGER NOT NULL,
    PRIMARY KEY (template_id, day_offset, meal, position)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS meal_plan_template_items;
DROP TABLE IF EXISTS meal_plan_templates;
-- +goose StatementEnd
