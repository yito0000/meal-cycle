package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/yito0000/meal-cycle/internal/domain/suggestion"
)

// SuggestionRepo は献立候補の算出に使う履歴を読む。書き込みは持たない。
type SuggestionRepo struct {
	db *sql.DB
}

func NewSuggestionRepo(db *sql.DB) *SuggestionRepo {
	return &SuggestionRepo{db: db}
}

// ListRecipeHistory は削除済みでないレシピごとに、作った印の付いた行の最も新しい日付と、today 以降に印の無い行があるかを返す
// (add-mealplan-suggest-by-recency design.md D2)。印は (batch_id, recipe_id) の組に付くので、作り置きの全行の日付に効く。
func (r *SuggestionRepo) ListRecipeHistory(ctx context.Context, today string) ([]suggestion.RecipeHistory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id,
		       (SELECT MAX(m.date) FROM meal_plans m
		          JOIN meal_plan_cooked c ON c.batch_id = m.batch_id AND c.recipe_id = m.recipe_id
		         WHERE m.recipe_id = r.id) AS last_cooked,
		       EXISTS (SELECT 1 FROM meal_plans m
		                WHERE m.recipe_id = r.id AND m.date >= ?
		                  AND NOT EXISTS (SELECT 1 FROM meal_plan_cooked c
		                                   WHERE c.batch_id = m.batch_id AND c.recipe_id = m.recipe_id)) AS upcoming
		FROM recipes r
		WHERE r.archived_at IS NULL
		ORDER BY r.id
	`, today)
	if err != nil {
		return nil, fmt.Errorf("query recipe history: %w", err)
	}
	defer rows.Close()

	out := make([]suggestion.RecipeHistory, 0)
	for rows.Next() {
		var (
			h    suggestion.RecipeHistory
			last sql.NullString
		)
		if err := rows.Scan(&h.RecipeID, &last, &h.HasUpcomingPlan); err != nil {
			return nil, fmt.Errorf("scan recipe history: %w", err)
		}
		h.LastCookedDate = last.String
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipe history: %w", err)
	}
	return out, nil
}
