-- +goose Up
-- +goose StatementBegin
ALTER TABLE meal_plans RENAME COLUMN slot TO meal;
-- +goose StatementEnd
