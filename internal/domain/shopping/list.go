// Package shopping は shopping capability (S07 買い物リスト) のドメイン層。
//
// 買い物リストは保存せず、献立と在庫から毎回算出する。量の突き合わせは近似であり、
// 結果は「期間の献立で使う材料の目安」として扱う (add-shopping-list design.md D5)。
package shopping

import "github.com/yito0000/meal-cycle/internal/domain/recipe"

// PlannedBatch は期間内で材料を 1 回分計上する単位 = (batch_id, recipe_id) の組。
//
// FirstDate / FirstMeal / FirstPosition は期間内で最初に出てくる行 (日付, 時間帯, セル内の並び) の値で、
// 並び順を S01 のセル内の順 (主菜 → 副菜 → 汁物) に揃えるために使う。Dates / Meals も期間内の行だけから求める。
// BatchMemberCount は期間外を含む batch_id 全体の行数で、S01 の batchMemberCount と同じ意味を持つ。
type PlannedBatch struct {
	FirstDate           string
	FirstMeal           string
	FirstPosition       int
	BatchID             string
	RecipeID            string
	RecipeName          string
	Dates               []string
	Meals               []string
	BatchMemberCount    int
	StartedBeforePeriod bool
	Cooked              bool
	Ingredients         []recipe.RecipeIngredient
}

// Dish は S07「この期間の献立」欄の 1 行。PlannedBatch から材料を除いたもの。
// BatchID と RecipeID の組で行を一意に識別できる。
type Dish struct {
	BatchID             string   `json:"batchId"`
	RecipeID            string   `json:"recipeId"`
	RecipeName          string   `json:"recipeName"`
	Dates               []string `json:"dates"`
	Meals               []string `json:"meals"`
	BatchMemberCount    int      `json:"batchMemberCount"`
	StartedBeforePeriod bool     `json:"startedBeforePeriod"`
	Cooked              bool     `json:"cooked"`
}

// StockAmount は突き合わせに使う在庫 1 行。期限切れの在庫は呼び出し側で除いておく。
type StockAmount struct {
	Name   string
	Amount float64
	Unit   string
}

// StockStatus は必要量に対する在庫の充足度。数値の内訳は見せず 3 段階に丸める。
type StockStatus string

const (
	StockNone    StockStatus = "none"
	StockPartial StockStatus = "partial"
	StockCovered StockStatus = "covered"
)

// Item は材料テーブルの 1 行。Amount は期間の必要量の合計 (在庫は引かない)。
type Item struct {
	Name        string      `json:"name"`
	Unit        string      `json:"unit"`
	Amount      float64     `json:"amount"`
	StockStatus StockStatus `json:"stockStatus"`
	Recipes     []string    `json:"recipes"`

	firstDate string
}

// List は ShoppingService.List の返り値。PeriodEnd は期間に含まれる最後の日。
type List struct {
	PeriodStart string `json:"periodStart"`
	PeriodEnd   string `json:"periodEnd"`
	PeriodDays  int    `json:"periodDays"`
	Dishes      []Dish `json:"dishes"`
	Items       []Item `json:"items"`
}
