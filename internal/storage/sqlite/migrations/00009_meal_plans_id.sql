-- +goose Up
-- +goose StatementBegin
-- 行を並び順ではなく ID で指すため、主キーを id に移す (add-mealplan-entry-id design.md D6)。
-- (date, meal, position) は「0 からの連番で欠番なし」を DB に守らせるため一意制約として残す (D2)。
CREATE TABLE meal_plans_new (
    id         TEXT NOT NULL PRIMARY KEY CHECK (length(id) > 0),
    date       TEXT NOT NULL,
    meal       TEXT NOT NULL,
    position   INTEGER NOT NULL,
    recipe_id  TEXT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    batch_id   TEXT NOT NULL CHECK (length(batch_id) > 0),
    UNIQUE (date, meal, position)
);
-- +goose StatementEnd

-- +goose StatementBegin
-- SQL だけでは UUID v7 を作れないため、既存行には無作為な hex を振る (D1)。ID は不透明な文字列として扱う。
INSERT INTO meal_plans_new (id, date, meal, position, recipe_id, created_at, batch_id)
SELECT lower(hex(randomblob(16))), date, meal, position, recipe_id, created_at, batch_id FROM meal_plans;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE meal_plans;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE meal_plans_new RENAME TO meal_plans;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_meal_plans_date ON meal_plans (date);
CREATE INDEX IF NOT EXISTS idx_meal_plans_batch_id ON meal_plans (batch_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE meal_plans_old (
    date       TEXT NOT NULL,
    meal       TEXT NOT NULL,
    position   INTEGER NOT NULL,
    recipe_id  TEXT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    batch_id   TEXT NOT NULL CHECK (length(batch_id) > 0),
    PRIMARY KEY (date, meal, position)
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO meal_plans_old (date, meal, position, recipe_id, created_at, batch_id)
SELECT date, meal, position, recipe_id, created_at, batch_id FROM meal_plans;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE meal_plans;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE meal_plans_old RENAME TO meal_plans;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_meal_plans_date ON meal_plans (date);
CREATE INDEX IF NOT EXISTS idx_meal_plans_batch_id ON meal_plans (batch_id);
-- +goose StatementEnd
