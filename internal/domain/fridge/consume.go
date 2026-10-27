package fridge

import (
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/yito0000/meal-cycle/internal/domain/ingredient"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

// ConsumptionCandidate は材料 1 つに対して減らす候補の在庫 1 行。Amount は減らす量の初期値。
type ConsumptionCandidate struct {
	StockID       string  `json:"stockId"`
	ExpiresAt     string  `json:"expiresAt"`
	AddedAt       string  `json:"addedAt"`
	CurrentAmount float64 `json:"currentAmount"`
	Amount        float64 `json:"amount"`
}

// ConsumptionLine はレシピの材料 1 つ (同じ食材名と単位は合算済み) と、その候補の在庫。
type ConsumptionLine struct {
	Name           string                 `json:"name"`
	Unit           string                 `json:"unit"`
	RequiredAmount float64                `json:"requiredAmount"`
	Candidates     []ConsumptionCandidate `json:"candidates"`
}

// ConsumptionPlan は「使った在庫を減らす」ダイアログの中身。Missing は候補の在庫が無い材料の食材名。
type ConsumptionPlan struct {
	Lines   []ConsumptionLine `json:"lines"`
	Missing []string          `json:"missing"`
}

// Consumption は在庫 1 行から減らす量。
type Consumption struct {
	StockID string  `json:"stockId"`
	Amount  float64 `json:"amount"`
}

// 表示と同じ小数第 2 位で丸め、0.1+0.2 の誤差で割り当てが在庫を超えないようにする。
func roundAmount(v float64) float64 {
	return math.Round(v*100) / 100
}

type consumeKey struct{ name, unit string }

// PlanConsumption はレシピの材料ごとに、減らす候補の在庫と減らす量の初期値を求める (add-mealplan-cook-consume design.md D3)。
// stocks は期限切れを除いた在庫を受け取る。食材名と単位は S07 と同じく ingredient.NormalizeName でそろえて比べる。
func PlanConsumption(ingredients []recipe.RecipeIngredient, stocks []StockItem, stapleNames []string, measuringUnits []string) ConsumptionPlan {
	staples := make(map[string]bool, len(stapleNames))
	for _, n := range stapleNames {
		staples[ingredient.NormalizeName(n)] = true
	}
	measuring := make([]string, 0, len(measuringUnits))
	for _, u := range measuringUnits {
		measuring = append(measuring, ingredient.NormalizeName(u))
	}

	index := map[consumeKey]int{}
	keys := make([]consumeKey, 0)
	lines := make([]ConsumptionLine, 0)
	for _, ing := range ingredients {
		k := consumeKey{ingredient.NormalizeName(ing.Name), ingredient.NormalizeName(ing.Unit)}
		if ing.Amount <= 0 || slices.Contains(measuring, k.unit) || staples[k.name] {
			continue
		}
		i, ok := index[k]
		if !ok {
			i = len(lines)
			index[k] = i
			keys = append(keys, k)
			lines = append(lines, ConsumptionLine{
				Name: strings.TrimSpace(ing.Name), Unit: strings.TrimSpace(ing.Unit), Candidates: []ConsumptionCandidate{},
			})
		}
		lines[i].RequiredAmount += ing.Amount
	}

	ordered := append([]StockItem(nil), stocks...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.ExpiresAt != b.ExpiresAt {
			if a.ExpiresAt == "" {
				return false
			}
			if b.ExpiresAt == "" {
				return true
			}
			return a.ExpiresAt < b.ExpiresAt
		}
		return a.AddedAt < b.AddedAt
	})

	plan := ConsumptionPlan{Lines: make([]ConsumptionLine, 0), Missing: make([]string, 0)}
	for i := range lines {
		line := lines[i]
		line.RequiredAmount = roundAmount(line.RequiredAmount)
		remaining := line.RequiredAmount
		for _, s := range ordered {
			if (consumeKey{ingredient.NormalizeName(s.Name), ingredient.NormalizeName(s.Unit)}) != keys[i] {
				continue
			}
			// 割り当てきれない分は捨てる。在庫は 0 より小さくならない (design.md Risks)。
			take := roundAmount(math.Min(remaining, s.Amount))
			remaining = roundAmount(remaining - take)
			line.Candidates = append(line.Candidates, ConsumptionCandidate{
				StockID: s.ID, ExpiresAt: s.ExpiresAt, AddedAt: s.AddedAt, CurrentAmount: s.Amount, Amount: take,
			})
		}
		if len(line.Candidates) == 0 {
			plan.Missing = append(plan.Missing, line.Name)
			continue
		}
		plan.Lines = append(plan.Lines, line)
	}
	return plan
}
