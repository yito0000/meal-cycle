package suggestion

import (
	"sort"
	"strings"
	"time"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/domain/ingredient"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

// ByStock は期限が 🔴 / 🟡 の在庫と食材名が同じ材料を持つレシピを「使い切り」の候補にする (add-mealplan-suggest-by-stock design.md D3)。
// 単位と量は比べない。候補の目的は在庫を使う料理を思い出させることで、使い切れるかの判定ではないため。
func ByStock(recipes []recipe.Recipe, stocks []fridge.StockItem, today time.Time, red, yellow, limit int) []Candidate {
	// 同じ食材名の在庫は最も近い期限の 1 件にまとめる。
	nearest := map[string]StockReason{}
	for _, s := range stocks {
		status, days := fridge.Evaluate(s.ExpiresAt, today, red, yellow)
		if (status != fridge.ExpiryUrgent && status != fridge.ExpirySoon) || days == nil {
			continue
		}
		key := ingredient.NormalizeName(s.Name)
		if cur, ok := nearest[key]; !ok || *days < cur.DaysLeft {
			nearest[key] = StockReason{Name: strings.TrimSpace(s.Name), DaysLeft: *days}
		}
	}

	out := make([]Candidate, 0)
	if len(nearest) == 0 {
		return out
	}
	for _, r := range recipes {
		if r.ArchivedAt != "" {
			continue
		}
		seen := map[string]bool{}
		reasons := make([]StockReason, 0)
		for _, ing := range r.Ingredients {
			key := ingredient.NormalizeName(ing.Name)
			reason, ok := nearest[key]
			if !ok || seen[key] {
				continue
			}
			seen[key] = true
			reasons = append(reasons, reason)
		}
		if len(reasons) == 0 {
			continue
		}
		sort.SliceStable(reasons, func(i, j int) bool { return reasons[i].DaysLeft < reasons[j].DaysLeft })
		out = append(out, Candidate{RecipeID: r.ID, RecipeName: r.Name, Stocks: reasons})
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if len(a.Stocks) != len(b.Stocks) {
			return len(a.Stocks) > len(b.Stocks)
		}
		if a.Stocks[0].DaysLeft != b.Stocks[0].DaysLeft {
			return a.Stocks[0].DaysLeft < b.Stocks[0].DaysLeft
		}
		return a.RecipeName < b.RecipeName
	})
	if limit >= 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
