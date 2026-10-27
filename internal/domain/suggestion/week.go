package suggestion

import "github.com/yito0000/meal-cycle/internal/domain/recipe"

// WeekSlot は週の候補で埋める空いたセル 1 つ。Meal は mealplan.Meal と同じ文字列。
type WeekSlot struct {
	Date string `json:"date"`
	Meal string `json:"meal"`
}

// WeekPlanItem は案の中の作り置き 1 つ。根拠は、そのレシピが当てはまる軸のものだけを埋める。
type WeekPlanItem struct {
	RecipeID   string        `json:"recipeId"`
	RecipeName string        `json:"recipeName"`
	Dates      []string      `json:"dates"`
	Meal       string        `json:"meal"`
	Stocks     []StockReason `json:"stocks"`
	DaysSince  int           `json:"daysSince"`
}

// WeekPlanKind は案の種類。表示名はフロントが i18n で引く。
type WeekPlanKind string

const (
	WeekPlanStock    WeekPlanKind = "stock"
	WeekPlanRecency  WeekPlanKind = "recency"
	WeekPlanBalanced WeekPlanKind = "balanced"
)

type WeekPlan struct {
	Kind  WeekPlanKind   `json:"kind"`
	Items []WeekPlanItem `json:"items"`
}

// WeekPlanResult は SuggestionService.WeekPlans の返り値。Slots が空なら埋めるコマが無い。
type WeekPlanResult struct {
	Slots []WeekSlot `json:"slots"`
	Plans []WeekPlan `json:"plans"`
}

// SplitSlots は日付の順に並んだコマを、組の数を最小にしたうえで大きさを均した組に分ける (add-mealplan-suggest-week-preset design.md D2)。
// 上限まで詰めて余りを最後に置くと、4 コマで 1 日だけの組ができ、作り置きの形から外れるため。
func SplitSlots(slots []WeekSlot, maxRunDays int) [][]WeekSlot {
	n := len(slots)
	out := make([][]WeekSlot, 0)
	if n == 0 || maxRunDays < 1 {
		return out
	}
	k := (n + maxRunDays - 1) / maxRunDays
	base, extra := n/k, n%k
	i := 0
	for g := 0; g < k; g++ {
		size := base
		if g < extra {
			size++
		}
		out = append(out, append([]WeekSlot(nil), slots[i:i+size]...))
		i += size
	}
	return out
}

// WeekPlans は組にレシピを 1 つずつ当てた案を、使い切り優先、久しぶり優先、バランスの順に返す (add-mealplan-suggest-week-preset design.md D3)。
// 乱数は使わない。押し直しても同じ案が出ることで、案の根拠 (順位表) が利用者から見て一貫するため。
// byStock / byRecency / rest は、削除済みと今日以降に予定のあるレシピを除いたものを、それぞれの順位で渡す。
func WeekPlans(groups [][]WeekSlot, byStock, byRecency []Candidate, rest []recipe.Recipe) []WeekPlan {
	out := make([]WeekPlan, 0)
	if len(groups) == 0 {
		return out
	}
	stockOf := map[string][]StockReason{}
	for _, c := range byStock {
		stockOf[c.RecipeID] = c.Stocks
	}
	daysOf := map[string]int{}
	for _, c := range byRecency {
		daysOf[c.RecipeID] = c.DaysSince
	}
	names := map[string]string{}
	toIDs := func(cs []Candidate) []string {
		ids := make([]string, 0, len(cs))
		for _, c := range cs {
			names[c.RecipeID] = c.RecipeName
			ids = append(ids, c.RecipeID)
		}
		return ids
	}
	stockIDs, recencyIDs := toIDs(byStock), toIDs(byRecency)
	restIDs := make([]string, 0, len(rest))
	for _, r := range rest {
		names[r.ID] = r.Name
		restIDs = append(restIDs, r.ID)
	}

	interleaved := make([]string, 0, len(stockIDs)+len(recencyIDs))
	for i := 0; i < len(stockIDs) || i < len(recencyIDs); i++ {
		if i < len(stockIDs) {
			interleaved = append(interleaved, stockIDs[i])
		}
		if i < len(recencyIDs) {
			interleaved = append(interleaved, recencyIDs[i])
		}
	}
	rankings := []struct {
		kind  WeekPlanKind
		order []string
	}{
		{WeekPlanStock, concat(stockIDs, recencyIDs, restIDs)},
		{WeekPlanRecency, concat(recencyIDs, stockIDs, restIDs)},
		{WeekPlanBalanced, concat(interleaved, restIDs)},
	}

	seen := map[string]bool{}
	for _, rk := range rankings {
		used := map[string]bool{}
		next := 0
		items := make([]WeekPlanItem, 0, len(groups))
		signature := ""
		for _, g := range groups {
			for next < len(rk.order) && used[rk.order[next]] {
				next++
			}
			// レシピが組の数より少なければ、当てられない組は案に含めない。
			if next >= len(rk.order) {
				break
			}
			id := rk.order[next]
			used[id] = true
			dates := make([]string, 0, len(g))
			for _, s := range g {
				dates = append(dates, s.Date)
			}
			stocks := stockOf[id]
			if stocks == nil {
				stocks = []StockReason{}
			}
			items = append(items, WeekPlanItem{
				RecipeID: id, RecipeName: names[id], Dates: dates, Meal: g[0].Meal, Stocks: stocks, DaysSince: daysOf[id],
			})
			signature += id + "\x00"
		}
		if len(items) == 0 || seen[signature] {
			continue
		}
		seen[signature] = true
		out = append(out, WeekPlan{Kind: rk.kind, Items: items})
	}
	return out
}

// concat は先に出た ID を優先し、重複を除いてつなげる。
func concat(lists ...[]string) []string {
	out := make([]string, 0)
	seen := map[string]bool{}
	for _, l := range lists {
		for _, id := range l {
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	return out
}
