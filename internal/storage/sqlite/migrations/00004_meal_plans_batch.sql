-- +goose Up
-- +goose StatementBegin
ALTER TABLE meal_plans ADD COLUMN batch_id TEXT;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE meal_plans SET batch_id = lower(hex(randomblob(16))) WHERE batch_id IS NULL OR batch_id = '';
-- +goose StatementEnd

-- +goose StatementBegin
-- 既存 index を残しつつバッチ検索用の index を追加。SQLite は ALTER で NOT NULL を後付け
-- できないため、実効的な NOT NULL 保証はアプリ側 (Assign / CopyLeftover / Split / Merge)
-- で必ず batch_id を書き込むことで行う。CHECK 制約で空文字だけは弾く。
CREATE INDEX IF NOT EXISTS idx_meal_plans_batch_id ON meal_plans (batch_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- SQLite は CHECK 制約の後付けを直接サポートしないため、テーブルを一旦作り直す。
-- CREATE TABLE ... AS SELECT でデータをコピーし、旧テーブルを drop → rename する。
CREATE TABLE meal_plans_new (
    date       TEXT NOT NULL,
    slot       TEXT NOT NULL,
    position   INTEGER NOT NULL,
    recipe_id  TEXT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    batch_id   TEXT NOT NULL CHECK (length(batch_id) > 0),
    PRIMARY KEY (date, slot, position)
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO meal_plans_new (date, slot, position, recipe_id, created_at, batch_id)
SELECT date, slot, position, recipe_id, created_at, batch_id FROM meal_plans;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE meal_plans;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE meal_plans_new RENAME TO meal_plans;
-- +goose StatementEnd

-- +goose StatementBegin
-- 00003 で作った index は DROP TABLE と一緒に消えているので、両方張り直す。
CREATE INDEX IF NOT EXISTS idx_meal_plans_date ON meal_plans (date);
CREATE INDEX IF NOT EXISTS idx_meal_plans_batch_id ON meal_plans (batch_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_meal_plans_batch_id;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE meal_plans_old (
    date       TEXT NOT NULL,
    slot       TEXT NOT NULL,
    position   INTEGER NOT NULL,
    recipe_id  TEXT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    PRIMARY KEY (date, slot, position)
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO meal_plans_old (date, slot, position, recipe_id, created_at)
SELECT date, slot, position, recipe_id, created_at FROM meal_plans;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE meal_plans;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE meal_plans_old RENAME TO meal_plans;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_meal_plans_date ON meal_plans (date);
-- +goose StatementEnd
