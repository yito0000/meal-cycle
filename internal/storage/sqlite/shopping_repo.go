package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/yito0000/meal-cycle/internal/domain/recipe"
	"github.com/yito0000/meal-cycle/internal/domain/shopping"
)

// ShoppingRepo は買い物リストの算出に必要な献立と材料を読み出す。書き込みは持たない。
type ShoppingRepo struct {
	db *sql.DB
}

func NewShoppingRepo(db *sql.DB) *ShoppingRepo {
	return &ShoppingRepo{db: db}
}

type batchKey struct {
	batchID  string
	recipeID string
}

// ListPlannedBatches は期間 [periodStart, periodStart+periodDays) の献立を (batch_id, recipe_id) ごとにまとめて返す。
// 入れ替え (Replace) で 1 つの batch_id に別レシピが混ざりうるため、batch_id だけでは束ねない (design.md D3)。
func (r *ShoppingRepo) ListPlannedBatches(
	ctx context.Context, periodStart string, periodDays int,
) ([]shopping.PlannedBatch, error) {
	end, err := addDays(periodStart, periodDays)
	if err != nil {
		return nil, err
	}

	// 3 本のクエリを同じスナップショットで読むため read-only tx にまとめる。
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	batches, index, err := loadPlannedGroups(ctx, tx, periodStart, end)
	if err != nil {
		return nil, err
	}
	if err := loadPlannedDatesAndMeals(ctx, tx, periodStart, end, batches, index); err != nil {
		return nil, err
	}
	if err := loadPlannedIngredients(ctx, tx, periodStart, end, batches); err != nil {
		return nil, err
	}
	return batches, nil
}

func loadPlannedGroups(
	ctx context.Context, tx *sql.Tx, start, end string,
) ([]shopping.PlannedBatch, map[batchKey]int, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT m.batch_id, m.recipe_id, r.name, MIN(m.date) AS first_date,
		       (SELECT COUNT(*) FROM meal_plans a WHERE a.batch_id = m.batch_id) AS member_count,
		       EXISTS (
		           SELECT 1 FROM meal_plans p
		           WHERE p.batch_id = m.batch_id AND p.recipe_id = m.recipe_id AND p.date < ?
		       ) AS started_before
		FROM meal_plans m
		JOIN recipes r ON r.id = m.recipe_id
		WHERE m.date >= ? AND m.date < ?
		GROUP BY m.batch_id, m.recipe_id
		ORDER BY first_date, r.name
	`, start, start, end)
	if err != nil {
		return nil, nil, fmt.Errorf("query planned batches: %w", err)
	}
	defer rows.Close()

	out := make([]shopping.PlannedBatch, 0)
	index := map[batchKey]int{}
	for rows.Next() {
		var k batchKey
		var b shopping.PlannedBatch
		if err := rows.Scan(&k.batchID, &k.recipeID, &b.RecipeName, &b.FirstDate,
			&b.BatchMemberCount, &b.StartedBeforePeriod); err != nil {
			return nil, nil, fmt.Errorf("scan planned batch: %w", err)
		}
		b.BatchID = k.batchID
		b.RecipeID = k.recipeID
		b.Dates = []string{}
		b.Meals = []string{}
		b.Ingredients = []recipe.RecipeIngredient{}
		index[k] = len(out)
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate planned batches: %w", err)
	}
	return out, index, nil
}

func loadPlannedDatesAndMeals(
	ctx context.Context, tx *sql.Tx, start, end string,
	batches []shopping.PlannedBatch, index map[batchKey]int,
) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT batch_id, recipe_id, date, meal, position
		FROM meal_plans
		WHERE date >= ? AND date < ?
		ORDER BY date,
		         CASE meal WHEN 'breakfast' THEN 0 WHEN 'lunch' THEN 1 WHEN 'dinner' THEN 2 ELSE 3 END,
		         position
	`, start, end)
	if err != nil {
		return fmt.Errorf("query planned dates: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var k batchKey
		var date, meal string
		var position int
		if err := rows.Scan(&k.batchID, &k.recipeID, &date, &meal, &position); err != nil {
			return fmt.Errorf("scan planned date: %w", err)
		}
		i, ok := index[k]
		if !ok {
			continue
		}
		b := &batches[i]
		// 行は (日付, 時間帯, position) 順に来るので、組ごとの最初の行が並び順の基準になる。
		if len(b.Dates) == 0 {
			b.FirstMeal = meal
			b.FirstPosition = position
		}
		if n := len(b.Dates); n == 0 || b.Dates[n-1] != date {
			b.Dates = append(b.Dates, date)
		}
		if !containsMeal(b.Meals, meal) {
			b.Meals = append(b.Meals, meal)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate planned dates: %w", err)
	}
	for i := range batches {
		shopping.SortMeals(batches[i].Meals)
	}
	return nil
}

func loadPlannedIngredients(
	ctx context.Context, tx *sql.Tx, start, end string, batches []shopping.PlannedBatch,
) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT recipe_id, name, amount, unit
		FROM recipe_ingredients
		WHERE recipe_id IN (SELECT DISTINCT recipe_id FROM meal_plans WHERE date >= ? AND date < ?)
		ORDER BY recipe_id, position
	`, start, end)
	if err != nil {
		return fmt.Errorf("query planned ingredients: %w", err)
	}
	defer rows.Close()

	byRecipe := map[string][]recipe.RecipeIngredient{}
	for rows.Next() {
		var recipeID string
		var ing recipe.RecipeIngredient
		if err := rows.Scan(&recipeID, &ing.Name, &ing.Amount, &ing.Unit); err != nil {
			return fmt.Errorf("scan planned ingredient: %w", err)
		}
		byRecipe[recipeID] = append(byRecipe[recipeID], ing)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate planned ingredients: %w", err)
	}
	for i := range batches {
		if ings, ok := byRecipe[batches[i].RecipeID]; ok {
			batches[i].Ingredients = append([]recipe.RecipeIngredient{}, ings...)
		}
	}
	return nil
}

func containsMeal(meals []string, m string) bool {
	for _, x := range meals {
		if x == m {
			return true
		}
	}
	return false
}
