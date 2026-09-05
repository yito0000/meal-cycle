// Package recipe は recipe capability (S02..S04) のドメイン層。
//
// Recipe はユーザーが登録した料理定義で、S02 レシピ一覧と S04 レシピ
// 登録/編集画面、将来 mealplan エントリからの RecipeID 参照の 3 箇所で使う。
//
// このパッケージはフレームワーク非依存に保つ。Wails サービスと SQLite
// リポジトリはこの上に乗る。
package recipe

// RecipeIngredient は Recipe 1 品に紐づく材料 1 行。
//
// 本 proposal (add-recipe-edit) では食材マスタを持たないので、Name は
// 自由入力の文字列として保存する。冷蔵庫在庫との名寄せが必要になった
// 段階で、後続 proposal `add-ingredient-master` が Name を ingredient
// テーブルへの FK に置き換える migration を書く (design.md D2 / Migration
// Plan を参照)。
type RecipeIngredient struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

// Recipe は 1 品のレシピ全体。
//
// UpdatedAt / CreatedAt は "YYYY-MM-DD" 形式ではなく RFC3339 (UTC) の
// 文字列で持ち、SQLite storage と JSON 契約の両方で time.Time を引き込
// まずに往復できるようにする。CreatedAt は Repository 側で発行し、
// フロントは参照のみを想定する。
//
// IngredientCount は S02 一覧 (Repository.List) の projection 用途で、
// GetByID や Create/Update から返される Recipe では常に
// len(Ingredients) と一致するよう Repository 側でセットする。
// (詳細は design.md D1 / Risks 節を参照)
//
// Steps は Markdown 記法の単一テキスト (書き手側 UI の textarea の内容
// そのまま)。読み手側の描画は S03 詳細ドロワー proposal で追加する。
// 順序を DB 側で維持する必要がないため配列にしていない (design.md D2 /
// D10 参照)。
//
// Tags は結合済み表示文字列ではなく []string にして、区切り文字や
// バッジ描画をフロント側で制御可能にする。
type Recipe struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Servings        int                `json:"servings"`
	Tags            []string           `json:"tags"`
	Ingredients     []RecipeIngredient `json:"ingredients"`
	IngredientCount int                `json:"ingredientCount"`
	Steps           string             `json:"steps"`
	Notes           string             `json:"notes"`
	CreatedAt       string             `json:"createdAt"`
	UpdatedAt       string             `json:"updatedAt"`
}
