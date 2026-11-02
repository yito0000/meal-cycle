package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
)

// TemplateRepo は献立テンプレートを扱う。名前の検証と重複検査はサービス層で行う。
type TemplateRepo struct {
	db *sql.DB
}

func NewTemplateRepo(db *sql.DB) *TemplateRepo {
	return &TemplateRepo{db: db}
}

// Create はテンプレートと行を 1 トランザクションで保存する。ID と作成日時はここで決める。
func (r *TemplateRepo) Create(ctx context.Context, name string, items []mealplan.TemplateItem) (mealplan.Template, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return mealplan.Template{}, fmt.Errorf("generate template id: %w", err)
	}
	t := mealplan.Template{ID: id.String(), Name: name, CreatedAt: time.Now().UTC().Format(time.RFC3339), Items: items}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return mealplan.Template{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO meal_plan_templates (id, name, created_at) VALUES (?, ?, ?)`, t.ID, t.Name, t.CreatedAt); err != nil {
		return mealplan.Template{}, fmt.Errorf("insert template: %w", err)
	}
	for _, it := range items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO meal_plan_template_items (template_id, day_offset, meal, position, recipe_id, group_no)
			VALUES (?, ?, ?, ?, ?, ?)
		`, t.ID, it.DayOffset, string(it.Meal), it.Position, it.RecipeID, it.GroupNo); err != nil {
			return mealplan.Template{}, fmt.Errorf("insert template item: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return mealplan.Template{}, fmt.Errorf("commit tx: %w", err)
	}
	return t, nil
}

// List は作成日時の新しい順に、行 (レシピ名と削除済みかを添える) ごと返す。
func (r *TemplateRepo) List(ctx context.Context) ([]mealplan.Template, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, created_at FROM meal_plan_templates ORDER BY created_at DESC, rowid DESC`)
	if err != nil {
		return nil, fmt.Errorf("query templates: %w", err)
	}
	out := make([]mealplan.Template, 0)
	for rows.Next() {
		var t mealplan.Template
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan template: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate templates: %w", err)
	}
	// 行のクエリを流す前に閉じ、接続を 2 本使わないようにする。
	_ = rows.Close()
	for i := range out {
		items, err := r.items(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Items = items
	}
	return out, nil
}

func (r *TemplateRepo) Get(ctx context.Context, id string) (mealplan.Template, error) {
	var t mealplan.Template
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, created_at FROM meal_plan_templates WHERE id = ?`, id).Scan(&t.ID, &t.Name, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return mealplan.Template{}, mealplan.ErrTemplateNotFound
	}
	if err != nil {
		return mealplan.Template{}, fmt.Errorf("query template: %w", err)
	}
	items, err := r.items(ctx, id)
	if err != nil {
		return mealplan.Template{}, err
	}
	t.Items = items
	return t, nil
}

func (r *TemplateRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM meal_plan_templates WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete template rows affected: %w", err)
	}
	if n == 0 {
		return mealplan.ErrTemplateNotFound
	}
	return nil
}

// ExistsName は前後の空白を除いて同じ名前のテンプレートがあるかを返す。
func (r *TemplateRepo) ExistsName(ctx context.Context, name string) (bool, error) {
	var n int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM meal_plan_templates WHERE trim(name) = ?`, strings.TrimSpace(name)).Scan(&n); err != nil {
		return false, fmt.Errorf("count template name: %w", err)
	}
	return n > 0, nil
}

func (r *TemplateRepo) items(ctx context.Context, templateID string) ([]mealplan.TemplateItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT i.day_offset, i.meal, i.position, i.recipe_id, i.group_no, rc.name, rc.archived_at IS NOT NULL
		FROM meal_plan_template_items i
		JOIN recipes rc ON rc.id = i.recipe_id
		WHERE i.template_id = ?
		ORDER BY i.day_offset,
		         CASE i.meal WHEN 'breakfast' THEN 0 WHEN 'lunch' THEN 1 WHEN 'dinner' THEN 2 ELSE 3 END,
		         i.position
	`, templateID)
	if err != nil {
		return nil, fmt.Errorf("query template items: %w", err)
	}
	defer rows.Close()
	out := make([]mealplan.TemplateItem, 0)
	for rows.Next() {
		var (
			it   mealplan.TemplateItem
			meal string
		)
		if err := rows.Scan(&it.DayOffset, &meal, &it.Position, &it.RecipeID, &it.GroupNo, &it.RecipeName, &it.Archived); err != nil {
			return nil, fmt.Errorf("scan template item: %w", err)
		}
		it.Meal = mealplan.Meal(meal)
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate template items: %w", err)
	}
	return out, nil
}
