// Package mealplan は献立カレンダー (S01) のドメイン層。
//
// MealPlan は「1 行 = 1 品」で保存する。同じ (date, slot) に 0〜N 個の
// Entry を並べることで主菜＋副菜＋汁物を表現する (詳細は docs/design.md §4)。
package mealplan

// Slot は 1 日のうちの区分。文字列値は SQLite カラムと一致させるため以後変えない。
type Slot string

const (
	SlotBreakfast Slot = "breakfast"
	SlotLunch     Slot = "lunch"
	SlotDinner    Slot = "dinner"
)

// Entry は (Date, Slot) の Position 位置に置かれる 1 品。
//
// Date は "YYYY-MM-DD" (ローカル日付、タイムゾーンなし)。Position は 0 起点で
// (Date, Slot) 内で一意、S01 セル内の縦の並び順 (主菜 → 副菜 → 汁物) を決める。
// RecipeID は永続化の主参照で、RecipeName は表示用に recipes と JOIN した値。
// BatchID は「同じバッチ = 1 回分の調理を共有」の識別子。BatchMemberCount は
// 同じ BatchID を持つ行の全体数 (repo が COUNT(*) OVER (PARTITION BY batch_id) で返す)。
type Entry struct {
	Date             string `json:"date"`
	Slot             Slot   `json:"slot"`
	Position         int    `json:"position"`
	RecipeID         string `json:"recipeId"`
	RecipeName       string `json:"recipeName"`
	BatchID          string `json:"batchId"`
	BatchMemberCount int    `json:"batchMemberCount"`
}

// BatchSummary は合流候補ダイアログに表示する「同 slot 内の他バッチ」の 1 件。
// 起点行 (代表) の日付・レシピ ID・レシピ名と、バッチ全体のメンバー数を持つ。
type BatchSummary struct {
	BatchID     string `json:"batchId"`
	Date        string `json:"date"`
	Slot        Slot   `json:"slot"`
	RecipeID    string `json:"recipeId"`
	RecipeName  string `json:"recipeName"`
	MemberCount int    `json:"memberCount"`
}
