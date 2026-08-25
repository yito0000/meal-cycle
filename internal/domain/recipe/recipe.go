// Package recipe は recipe capability (S02..S04) のドメイン層。
//
// Recipe はユーザーが登録した料理定義で、S02 レシピ一覧と、将来 mealplan
// エントリからの RecipeID 参照の両方で使う。本 proposal (add-recipe-listing)
// では一覧表示に必要な最小フィールドだけを定義し、Ingredients / Steps / Notes
// は add-recipe-detail-drawer / add-recipe-crud で追加する。
//
// このパッケージはフレームワーク非依存に保つ。Wails サービスと SQLite
// リポジトリ (後続 proposal で追加) はこの上に乗る。
package recipe

// Recipe は S02 一覧向けに射影した 1 品のレシピ。
//
// UpdatedAt は "YYYY-MM-DD" 文字列 (ローカル日付、タイムゾーンなし) で
// 持ち、fixture と将来の SQLite storage の間で JSON 契約に time.Time を
// 引き込まずに往復できるようにする。
//
// Tags は結合済み表示文字列ではなく []string にして、区切り文字や将来の
// バッジ描画をフロント側で制御可能にする。詳細は design.md D5 を参照。
type Recipe struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Tags            []string `json:"tags"`
	IngredientCount int      `json:"ingredientCount"`
	UpdatedAt       string   `json:"updatedAt"`
}
