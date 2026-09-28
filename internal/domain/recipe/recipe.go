// Package recipe は recipe capability (S02..S04) のドメイン層。
package recipe

// RecipeIngredient は 1 レシピに紐づく材料 1 行。
// 本 proposal では食材マスタを持たないので Name は自由入力の文字列。将来
// add-ingredient-master で FK 参照に切り替える (design.md D2)。
type RecipeIngredient struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

// Recipe は 1 品のレシピ全体。
//
// UpdatedAt / CreatedAt は RFC3339 (UTC) 文字列。SQLite と JSON 契約の
// 両方で time.Time を引き込まずに往復するため。CreatedAt は Repository が
// 発行し、フロントは参照のみ。
//
// IngredientCount は S02 一覧 (Repository.List) の projection 用途。GetByID /
// Create / Update から返される Recipe では常に len(Ingredients) と一致するよう
// Repository でセットする (design.md D1)。
//
// Steps は Markdown 記法の単一テキスト (S04 textarea の内容そのまま)。読み手側
// の描画は S03 詳細ドロワー proposal で追加する (design.md D10)。
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
	ArchivedAt      string             `json:"archivedAt"`
}
