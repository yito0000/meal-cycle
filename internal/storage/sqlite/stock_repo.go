package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
)

// StockRepo は stock_items を扱う。ExpiryStatus / DaysLeft は日付に依存する算出値
// なので Repository では埋めず、Service 層で載せる (design.md D4)。
type StockRepo struct {
	db *sql.DB
}

func NewStockRepo(db *sql.DB) *StockRepo {
	return &StockRepo{db: db}
}

const stockColumns = `id, name, amount, unit, expires_at, added_at, updated_at`

func (r *StockRepo) List(ctx context.Context) ([]fridge.StockItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+stockColumns+`
		FROM stock_items
		ORDER BY added_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query stock_items: %w", err)
	}
	defer rows.Close()

	out := make([]fridge.StockItem, 0)
	for rows.Next() {
		item, err := scanStockItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stock_items: %w", err)
	}
	return out, nil
}

func (r *StockRepo) GetByID(ctx context.Context, id string) (fridge.StockItem, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+stockColumns+` FROM stock_items WHERE id = ?`, id)
	item, err := scanStockItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return fridge.StockItem{}, fridge.ErrStockNotFound
	}
	return item, err
}

// Create は ID / AddedAt / UpdatedAt を Repository 側で発行する。
func (r *StockRepo) Create(ctx context.Context, input fridge.StockItem) (fridge.StockItem, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return fridge.StockItem{}, fmt.Errorf("generate stock id: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)

	item := stripDerived(input)
	item.ID = id.String()
	item.AddedAt = now
	item.UpdatedAt = now
	if err := insertStockItem(ctx, r.db, item); err != nil {
		return fridge.StockItem{}, err
	}
	return item, nil
}

// Update は added_at を保ったまま他の列を上書きする。
func (r *StockRepo) Update(ctx context.Context, id string, input fridge.StockItem) (fridge.StockItem, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.ExecContext(ctx, `
		UPDATE stock_items
		SET name = ?, amount = ?, unit = ?, expires_at = ?, updated_at = ?
		WHERE id = ?
	`, input.Name, input.Amount, input.Unit, nullableDate(input.ExpiresAt), now, id)
	if err != nil {
		return fridge.StockItem{}, fmt.Errorf("update stock_item: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fridge.StockItem{}, fmt.Errorf("update stock_item rows affected: %w", err)
	}
	if affected == 0 {
		return fridge.StockItem{}, fridge.ErrStockNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *StockRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM stock_items WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete stock_item: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete stock_item rows affected: %w", err)
	}
	if affected == 0 {
		return fridge.ErrStockNotFound
	}
	return nil
}

// Restore は削除直後の「元に戻す」用に、渡された ID / AddedAt のまま行を作り直す。
// トーストの二重押しで同じ ID が来たら ErrStockAlreadyExists (design.md D8)。
func (r *StockRepo) Restore(ctx context.Context, input fridge.StockItem) (fridge.StockItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fridge.StockItem{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var exists int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM stock_items WHERE id = ?`, input.ID).Scan(&exists)
	if err == nil {
		return fridge.StockItem{}, fridge.ErrStockAlreadyExists
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fridge.StockItem{}, fmt.Errorf("check stock_item exists: %w", err)
	}

	item := stripDerived(input)
	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := insertStockItem(ctx, tx, item); err != nil {
		return fridge.StockItem{}, err
	}
	if err := tx.Commit(); err != nil {
		return fridge.StockItem{}, fmt.Errorf("commit tx: %w", err)
	}
	return item, nil
}

// ListSuggestions は在庫名とレシピ材料名を合わせた食材名候補を返す。
// 名前ごとの単位は「在庫の added_at 最新行」→「レシピの updated_at 最新の材料行」の順で
// 決める (design.md D5)。recipe_ingredients は参照のみ。
func (r *StockRepo) ListSuggestions(ctx context.Context) (fridge.Suggestions, error) {
	out := fridge.Suggestions{
		Names: make([]fridge.NameSuggestion, 0),
		Units: make([]string, 0),
	}

	rows, err := r.db.QueryContext(ctx, `
		WITH candidates AS (
			SELECT name, unit, 0 AS src, added_at AS ts, 0 AS pos FROM stock_items
			UNION ALL
			SELECT ri.name, ri.unit, 1 AS src, r.updated_at AS ts, ri.position AS pos
			FROM recipe_ingredients ri
			JOIN recipes r ON r.id = ri.recipe_id
		),
		ranked AS (
			SELECT name, unit,
			       ROW_NUMBER() OVER (PARTITION BY name ORDER BY src ASC, ts DESC, pos ASC) AS rn
			FROM candidates
			WHERE name <> ''
		)
		SELECT name, unit FROM ranked WHERE rn = 1 ORDER BY name
	`)
	if err != nil {
		return fridge.Suggestions{}, fmt.Errorf("query name suggestions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s fridge.NameSuggestion
		if err := rows.Scan(&s.Name, &s.Unit); err != nil {
			return fridge.Suggestions{}, fmt.Errorf("scan name suggestion: %w", err)
		}
		out.Names = append(out.Names, s)
	}
	if err := rows.Err(); err != nil {
		return fridge.Suggestions{}, fmt.Errorf("iterate name suggestions: %w", err)
	}

	unitRows, err := r.db.QueryContext(ctx, `
		SELECT unit FROM stock_items WHERE unit <> ''
		UNION
		SELECT unit FROM recipe_ingredients WHERE unit <> ''
		ORDER BY unit
	`)
	if err != nil {
		return fridge.Suggestions{}, fmt.Errorf("query unit suggestions: %w", err)
	}
	defer unitRows.Close()
	for unitRows.Next() {
		var u string
		if err := unitRows.Scan(&u); err != nil {
			return fridge.Suggestions{}, fmt.Errorf("scan unit suggestion: %w", err)
		}
		out.Units = append(out.Units, u)
	}
	if err := unitRows.Err(); err != nil {
		return fridge.Suggestions{}, fmt.Errorf("iterate unit suggestions: %w", err)
	}
	return out, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// expires_at の NULL ⇔ "" 変換は scanStockItem / nullableDate の 2 関数に閉じる (design.md D3)。
func scanStockItem(s rowScanner) (fridge.StockItem, error) {
	var (
		item      fridge.StockItem
		expiresAt sql.NullString
	)
	if err := s.Scan(
		&item.ID, &item.Name, &item.Amount, &item.Unit, &expiresAt, &item.AddedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fridge.StockItem{}, err
		}
		return fridge.StockItem{}, fmt.Errorf("scan stock_item: %w", err)
	}
	item.ExpiresAt = expiresAt.String
	return item, nil
}

func nullableDate(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func insertStockItem(ctx context.Context, e execer, item fridge.StockItem) error {
	_, err := e.ExecContext(ctx, `
		INSERT INTO stock_items (`+stockColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.Name, item.Amount, item.Unit, nullableDate(item.ExpiresAt), item.AddedAt, item.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert stock_item: %w", err)
	}
	return nil
}

func stripDerived(item fridge.StockItem) fridge.StockItem {
	item.ExpiryStatus = ""
	item.DaysLeft = nil
	return item
}
