package suggestion

import (
	"sort"
	"time"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

// ByRecency は最終調理日から thresholdDays 日以上たったレシピを「久しぶりに」の候補にする (add-mealplan-suggest-by-recency design.md D1 / D3 / D4)。
// 一度も作っていないレシピと、今日以降に印の無い予定があるレシピは除く。予定のあるレシピを出すと二重の割り当てを誘うため。
func ByRecency(recipes []recipe.Recipe, history []RecipeHistory, today time.Time, thresholdDays, limit int) []Candidate {
	byID := make(map[string]RecipeHistory, len(history))
	for _, h := range history {
		byID[h.RecipeID] = h
	}
	y, m, d := today.Date()
	todayDate := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

	out := make([]Candidate, 0)
	for _, r := range recipes {
		if r.ArchivedAt != "" {
			continue
		}
		h, ok := byID[r.ID]
		if !ok || h.LastCookedDate == "" || h.HasUpcomingPlan {
			continue
		}
		last, err := time.Parse(fridge.DateLayout, h.LastCookedDate)
		if err != nil {
			continue
		}
		// 暦日同士の差で数え、夏時間などで 1 日が 24h でない地域でもずれないようにする。
		days := int(todayDate.Sub(last).Hours() / 24)
		if days < thresholdDays {
			continue
		}
		out = append(out, Candidate{RecipeID: r.ID, RecipeName: r.Name, Stocks: []StockReason{}, DaysSince: days})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DaysSince != out[j].DaysSince {
			return out[i].DaysSince > out[j].DaysSince
		}
		return out[i].RecipeName < out[j].RecipeName
	})
	if limit >= 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
