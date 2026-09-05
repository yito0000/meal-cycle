-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS recipes (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    servings   INTEGER NOT NULL DEFAULT 1,
    tags_json  TEXT NOT NULL DEFAULT '[]',
    steps      TEXT NOT NULL DEFAULT '',
    notes      TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS recipe_ingredients (
    recipe_id TEXT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    name      TEXT NOT NULL,
    amount    REAL NOT NULL,
    unit      TEXT NOT NULL,
    position  INTEGER NOT NULL,
    PRIMARY KEY (recipe_id, position)
);

CREATE INDEX IF NOT EXISTS idx_recipes_updated_at ON recipes (updated_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS recipe_ingredients;
DROP TABLE IF EXISTS recipes;
-- +goose StatementEnd
