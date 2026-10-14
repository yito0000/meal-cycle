package mealplan

// RemovedEntry は削除した 1 行の内容で、「元に戻す」で同じ行を作り直すために返す。
// 一覧の Entry と違い、作った印の日時と行の作成日時を持つ (revise-mealplan-destructive-actions design.md D1)。
type RemovedEntry struct {
	Date      string `json:"date"`
	Meal      Meal   `json:"meal"`
	Position  int    `json:"position"`
	RecipeID  string `json:"recipeId"`
	BatchID   string `json:"batchId"`
	CreatedAt string `json:"createdAt"`
	// 空文字 = 印なし
	CookedAt string `json:"cookedAt"`
}
