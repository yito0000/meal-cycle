package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

// ErrRecipeNotFound は指定 ID のレシピが見つからないときに GetByID / Update が返す。
var ErrRecipeNotFound = errors.New("recipe not found")

// ErrRecipeNameEmpty は Create / Update が空文字の名前で呼ばれたときに返す。
// バリデーションは基本的に Service 層で行うが、Repository でも最終防衛線として弾く。
var ErrRecipeNameEmpty = errors.New("recipe name empty")

// RecipeRepo は recipes / recipe_ingredients テーブルに対する CRUD を扱う。
//
// Recipe と RecipeIngredient は 1 トランザクションで一括保存する。
// 本 proposal では食材マスタを持たないので `recipe_ingredients.name` に材料名を
// 直接保存する。将来 `add-ingredient-master` proposal で ingredient_id へ
// 置き換える migration を書く (design.md D2 / Migration Plan を参照)。
//
// List (S02 用) は summary 用途で Ingredients / Steps / Notes を空値で返し、
// GetByID / Create / Update のみが詳細フィールドを詰めた Recipe を返す。
type RecipeRepo struct {
	db *sql.DB
}

// NewRecipeRepo は RecipeRepo を db に紐付けて返す。
func NewRecipeRepo(db *sql.DB) *RecipeRepo {
	return &RecipeRepo{db: db}
}

// List は登録済み全レシピを updated_at 降順で返す。
//
// S02 一覧の projection 用途のため、Ingredients / Steps / Notes は空値で
// 返す。IngredientCount は recipe_ingredients の件数を LEFT JOIN で 1 クエリ
// に合成する。
func (r *RecipeRepo) List(ctx context.Context) ([]recipe.Recipe, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.name, r.servings, r.tags_json, r.notes, r.created_at, r.updated_at,
		       COALESCE(c.cnt, 0) AS ingredient_count
		FROM recipes r
		LEFT JOIN (
			SELECT recipe_id, COUNT(*) AS cnt
			FROM recipe_ingredients
			GROUP BY recipe_id
		) c ON c.recipe_id = r.id
		ORDER BY r.updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query recipes: %w", err)
	}
	defer rows.Close()

	out := make([]recipe.Recipe, 0)
	for rows.Next() {
		var (
			rec      recipe.Recipe
			tagsJSON string
		)
		if err := rows.Scan(
			&rec.ID, &rec.Name, &rec.Servings, &tagsJSON, &rec.Notes,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.IngredientCount,
		); err != nil {
			return nil, fmt.Errorf("scan recipe row: %w", err)
		}
		if err := json.Unmarshal([]byte(tagsJSON), &rec.Tags); err != nil {
			return nil, fmt.Errorf("decode tags_json: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipes: %w", err)
	}
	return out, nil
}

// GetByID は指定 ID のレシピを詳細フィールド込みで返す。該当なしは ErrRecipeNotFound。
func (r *RecipeRepo) GetByID(ctx context.Context, id string) (recipe.Recipe, error) {
	var (
		rec      recipe.Recipe
		tagsJSON string
	)
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, servings, tags_json, steps, notes, created_at, updated_at
		FROM recipes
		WHERE id = ?
	`, id).Scan(
		&rec.ID, &rec.Name, &rec.Servings, &tagsJSON, &rec.Steps,
		&rec.Notes, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return recipe.Recipe{}, ErrRecipeNotFound
	}
	if err != nil {
		return recipe.Recipe{}, fmt.Errorf("query recipe by id: %w", err)
	}
	if err := json.Unmarshal([]byte(tagsJSON), &rec.Tags); err != nil {
		return recipe.Recipe{}, fmt.Errorf("decode tags_json: %w", err)
	}

	ings, err := r.loadIngredients(ctx, id)
	if err != nil {
		return recipe.Recipe{}, err
	}
	rec.Ingredients = ings
	rec.IngredientCount = len(ings)
	return rec, nil
}

// Create は新規レシピを保存し、Repository で決定した ID / CreatedAt / UpdatedAt を含めて返す。
func (r *RecipeRepo) Create(ctx context.Context, input recipe.Recipe) (recipe.Recipe, error) {
	if input.Name == "" {
		return recipe.Recipe{}, ErrRecipeNameEmpty
	}
	id, err := uuid.NewV7()
	if err != nil {
		return recipe.Recipe{}, fmt.Errorf("generate recipe id: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)

	rec := input
	rec.ID = id.String()
	rec.CreatedAt = now
	rec.UpdatedAt = now
	rec.IngredientCount = len(rec.Ingredients)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return recipe.Recipe{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := insertRecipeRow(ctx, tx, rec); err != nil {
		return recipe.Recipe{}, err
	}
	if err := insertRecipeIngredients(ctx, tx, rec.ID, rec.Ingredients); err != nil {
		return recipe.Recipe{}, err
	}
	if err := tx.Commit(); err != nil {
		return recipe.Recipe{}, fmt.Errorf("commit tx: %w", err)
	}
	return rec, nil
}

// Update は既存レシピを新しい内容で上書きする。updated_at は現在時刻に更新される。
// 該当 ID が無ければ ErrRecipeNotFound。name が空なら ErrRecipeNameEmpty。
func (r *RecipeRepo) Update(ctx context.Context, id string, input recipe.Recipe) (recipe.Recipe, error) {
	if input.Name == "" {
		return recipe.Recipe{}, ErrRecipeNameEmpty
	}
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return recipe.Recipe{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	// updated_at を必ず進めるため、time.Now が既存 updated_at と同一秒に
	// 落ちたときは +1s する (RFC3339 秒精度)。
	if now == existing.UpdatedAt {
		now = time.Now().UTC().Add(time.Second).Format(time.RFC3339)
	}

	rec := input
	rec.ID = id
	rec.CreatedAt = existing.CreatedAt
	rec.UpdatedAt = now
	rec.IngredientCount = len(rec.Ingredients)

	tagsJSON, err := json.Marshal(coalesceStrings(rec.Tags))
	if err != nil {
		return recipe.Recipe{}, fmt.Errorf("encode tags: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return recipe.Recipe{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `
		UPDATE recipes
		SET name = ?, servings = ?, tags_json = ?, steps = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`, rec.Name, rec.Servings, string(tagsJSON), rec.Steps, rec.Notes, rec.UpdatedAt, id)
	if err != nil {
		return recipe.Recipe{}, fmt.Errorf("update recipe: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return recipe.Recipe{}, fmt.Errorf("update recipe rows affected: %w", err)
	}
	if affected == 0 {
		return recipe.Recipe{}, ErrRecipeNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM recipe_ingredients WHERE recipe_id = ?`, id); err != nil {
		return recipe.Recipe{}, fmt.Errorf("delete recipe_ingredients: %w", err)
	}
	if err := insertRecipeIngredients(ctx, tx, id, rec.Ingredients); err != nil {
		return recipe.Recipe{}, err
	}
	if err := tx.Commit(); err != nil {
		return recipe.Recipe{}, fmt.Errorf("commit tx: %w", err)
	}
	return rec, nil
}

// insertRecipeRow は 1 件の recipe を tx で書き込む。
func insertRecipeRow(ctx context.Context, tx *sql.Tx, rec recipe.Recipe) error {
	tagsJSON, err := json.Marshal(coalesceStrings(rec.Tags))
	if err != nil {
		return fmt.Errorf("encode tags: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO recipes (id, name, servings, tags_json, steps, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		rec.ID, rec.Name, rec.Servings, string(tagsJSON), rec.Steps,
		rec.Notes, rec.CreatedAt, rec.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert recipe: %w", err)
	}
	return nil
}

// insertRecipeIngredients は recipe_ingredients を入力順に position で保存する。
func insertRecipeIngredients(ctx context.Context, tx *sql.Tx, recipeID string, ings []recipe.RecipeIngredient) error {
	for pos, ing := range ings {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO recipe_ingredients (recipe_id, name, amount, unit, position)
			VALUES (?, ?, ?, ?, ?)
		`, recipeID, ing.Name, ing.Amount, ing.Unit, pos); err != nil {
			return fmt.Errorf("insert recipe_ingredient (pos=%d): %w", pos, err)
		}
	}
	return nil
}

// loadIngredients は 1 件の recipe に紐づく材料行を position 順で返す。
func (r *RecipeRepo) loadIngredients(ctx context.Context, recipeID string) ([]recipe.RecipeIngredient, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT name, amount, unit
		FROM recipe_ingredients
		WHERE recipe_id = ?
		ORDER BY position
	`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("query recipe_ingredients: %w", err)
	}
	defer rows.Close()

	out := make([]recipe.RecipeIngredient, 0)
	for rows.Next() {
		var ri recipe.RecipeIngredient
		if err := rows.Scan(&ri.Name, &ri.Amount, &ri.Unit); err != nil {
			return nil, fmt.Errorf("scan recipe_ingredient: %w", err)
		}
		out = append(out, ri)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipe_ingredients: %w", err)
	}
	return out, nil
}

// coalesceStrings は nil を空スライスに置き換えて JSON エンコード結果を "[]" にする。
func coalesceStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
