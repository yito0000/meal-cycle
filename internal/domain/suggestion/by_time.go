package suggestion

import (
	"sort"

	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

// ByTime は所要時間が maxMinutes 以下のレシピを「時短」の候補にする。所要時間の無いレシピは根拠が無いので出さない
// (add-mealplan-suggest-by-time design.md D1 / D3)。閾値で切るのは、長いレシピが「時短」の欄に出て根拠と名前が合わなくなるのを避けるため。
func ByTime(recipes []recipe.Recipe, maxMinutes, limit int) []Candidate {
	out := make([]Candidate, 0)
	for _, r := range recipes {
		if r.ArchivedAt != "" || r.CookMinutes <= 0 || r.CookMinutes > maxMinutes {
			continue
		}
		out = append(out, Candidate{RecipeID: r.ID, RecipeName: r.Name, Stocks: []StockReason{}, CookMinutes: r.CookMinutes})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CookMinutes != out[j].CookMinutes {
			return out[i].CookMinutes < out[j].CookMinutes
		}
		return out[i].RecipeName < out[j].RecipeName
	})
	if limit >= 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
