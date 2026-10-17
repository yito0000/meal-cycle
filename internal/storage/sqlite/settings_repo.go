package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// SettingsRepo は settings を扱う。value の解釈と検証は Service 層で key ごとに行う (design.md D6)。
type SettingsRepo struct {
	db *sql.DB
}

func NewSettingsRepo(db *sql.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) Load(ctx context.Context) (map[string]json.RawMessage, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, fmt.Errorf("query settings: %w", err)
	}
	defer rows.Close()

	out := make(map[string]json.RawMessage)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan settings: %w", err)
		}
		out[key] = json.RawMessage(value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate settings: %w", err)
	}
	return out, nil
}

// ReplaceAll は rows に無い key を消す。既定値に戻した項目の行を残さないため。
func (r *SettingsRepo) ReplaceAll(ctx context.Context, rows map[string]json.RawMessage, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin settings tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM settings`); err != nil {
		return fmt.Errorf("delete settings: %w", err)
	}
	stamp := now.UTC().Format(time.RFC3339)
	for key, value := range rows {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)`, key, string(value), stamp,
		); err != nil {
			return fmt.Errorf("insert setting %s: %w", key, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit settings tx: %w", err)
	}
	return nil
}

func (r *SettingsRepo) DeleteAll(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM settings`); err != nil {
		return fmt.Errorf("delete settings: %w", err)
	}
	return nil
}
