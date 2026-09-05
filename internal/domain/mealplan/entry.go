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
// RecipeName は表示用の暫定フィールドで、add-mealplan-recipe-reference で
// RecipeID 参照に置き換える。
type Entry struct {
	Date       string `json:"date"`
	Slot       Slot   `json:"slot"`
	Position   int    `json:"position"`
	RecipeName string `json:"recipeName"`
}
