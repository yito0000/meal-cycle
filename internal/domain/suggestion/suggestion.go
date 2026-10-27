// Package suggestion は suggestion capability (S01 の献立候補) のドメイン層。
//
// 候補は全自動の割り当てではなく、根拠を添えて並べるだけにする (docs/product.md §5)。
// 根拠の文言は Go で組まず、フロントが i18n で組む (docs/design.md §7)。
package suggestion

// StockReason は「使い切り」の根拠となる在庫 1 件。DaysLeft は今日から賞味期限までの日数。
type StockReason struct {
	Name     string `json:"name"`
	DaysLeft int    `json:"daysLeft"`
}

// Candidate はレシピ候補 1 件。軸ごとの根拠は、その軸の欄でだけ使う。
// DaysSince は「久しぶりに」の根拠で、最終調理日から今日までの日数。
type Candidate struct {
	RecipeID   string        `json:"recipeId"`
	RecipeName string        `json:"recipeName"`
	Stocks     []StockReason `json:"stocks"`
	DaysSince  int           `json:"daysSince"`
}

// RecipeHistory はレシピ 1 件の調理履歴。LastCookedDate は作った印の付いた行の最も新しい日付 (YYYY-MM-DD) で、
// 一度も作っていなければ空。HasUpcomingPlan は今日以降に作った印の無い行があるか。
type RecipeHistory struct {
	RecipeID        string
	LastCookedDate  string
	HasUpcomingPlan bool
}

// Result は SuggestionService.List の返り値。軸ごとの候補を並べ、ダイアログを開いたときに 1 回で全軸を返す (add-mealplan-suggest-by-stock design.md D2)。
type Result struct {
	ByStock   []Candidate `json:"byStock"`
	ByRecency []Candidate `json:"byRecency"`
}
