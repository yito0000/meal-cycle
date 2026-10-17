// Package mealplan は献立カレンダー (S01) のドメイン層。
//
// MealPlan は「1 行 = 1 品」で保存する。同じ (date, meal) に 0〜N 個の
// Entry を並べることで主菜＋副菜＋汁物を表現する (詳細は docs/design.md §4)。
package mealplan

// Meal は 1 日のうちの区分。文字列値は SQLite カラムと一致させるため以後変えない。
type Meal string

const (
	MealBreakfast Meal = "breakfast"
	MealLunch     Meal = "lunch"
	MealDinner    Meal = "dinner"
)

// Entry は (Date, Meal) の Position 位置に置かれる 1 品。
//
// ID は行そのものの識別子で、1 品に対する操作はこれで行を指す。Position は並べ替えや削除のたびに
// 振り直されるので、行の指定には使わない (add-mealplan-entry-id)。
//
// Date は "YYYY-MM-DD" (ローカル日付、タイムゾーンなし)。Position は 0 起点で
// (Date, Meal) 内で一意、S01 セル内の縦の並び順 (主菜 → 副菜 → 汁物) を決める。
// RecipeID は永続化の主参照で、RecipeName は表示用に recipes と JOIN した値。
// BatchID は「同じバッチ = 1 回分の調理を共有」の識別子。BatchMemberCount は
// 同じ BatchID を持つ行の全体数 (表示期間の外の行も数える)。
// Cooked は同じ (BatchID, RecipeID) の組に作った印が付いているか。
type Entry struct {
	ID               string `json:"id"`
	Date             string `json:"date"`
	Meal             Meal   `json:"meal"`
	Position         int    `json:"position"`
	RecipeID         string `json:"recipeId"`
	RecipeName       string `json:"recipeName"`
	BatchID          string `json:"batchId"`
	BatchMemberCount int    `json:"batchMemberCount"`
	Cooked           bool   `json:"cooked"`
}

// BatchSummary は合流候補ダイアログに表示する「同 meal 内の他バッチ」の 1 件。
// 起点行 (代表) の日付・レシピ ID・レシピ名と、バッチ全体のメンバー数を持つ。
type BatchSummary struct {
	BatchID     string `json:"batchId"`
	Date        string `json:"date"`
	Meal        Meal   `json:"meal"`
	RecipeID    string `json:"recipeId"`
	RecipeName  string `json:"recipeName"`
	MemberCount int    `json:"memberCount"`
}
