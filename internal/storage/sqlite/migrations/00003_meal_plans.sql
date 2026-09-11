-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS meal_plans (
    date       TEXT NOT NULL,
    slot       TEXT NOT NULL,
    position   INTEGER NOT NULL,
    recipe_id  TEXT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    PRIMARY KEY (date, slot, position)
);

CREATE INDEX IF NOT EXISTS idx_meal_plans_date ON meal_plans (date);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS meal_plans;
-- +goose StatementEnd
