package mealplan

// PlanItem はまとめて割り当てる 1 行。同じ Group の行は 1 つの作り置き (batch_id) を共有する
// (add-mealplan-suggest-week-preset design.md D5)。
type PlanItem struct {
	Date     string `json:"date"`
	Meal     Meal   `json:"meal"`
	RecipeID string `json:"recipeId"`
	Group    int    `json:"group"`
}

// ValidatePlanItems は 1 行でも不正なら全体を拒む。途中まで割り当てた状態を作らないため。
func ValidatePlanItems(items []PlanItem) error {
	if len(items) == 0 {
		return ErrEmptyPlan
	}
	for _, it := range items {
		if err := ValidateDate(it.Date); err != nil {
			return err
		}
		if err := ValidateMeal(string(it.Meal)); err != nil {
			return err
		}
		if it.RecipeID == "" {
			return ErrUnknownRecipe
		}
	}
	return nil
}
