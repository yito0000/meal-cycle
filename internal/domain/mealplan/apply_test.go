package mealplan

import (
	"errors"
	"testing"
)

func TestValidatePlanItems(t *testing.T) {
	ok := PlanItem{Date: "2026-10-05", Meal: MealDinner, RecipeID: "r1", Group: 0}
	cases := []struct {
		name  string
		items []PlanItem
		want  error
	}{
		{"正しい行は通る", []PlanItem{ok}, nil},
		{"空の配列はエラーになる", []PlanItem{}, ErrEmptyPlan},
		{"実在しない日付はエラーになる", []PlanItem{ok, {Date: "2026-02-30", Meal: MealDinner, RecipeID: "r1"}}, ErrInvalidDate},
		{"未知の時間帯はエラーになる", []PlanItem{{Date: "2026-10-05", Meal: "brunch", RecipeID: "r1"}}, ErrInvalidMeal},
		{"レシピ ID が空の行はエラーになる", []PlanItem{{Date: "2026-10-05", Meal: MealDinner}}, ErrUnknownRecipe},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidatePlanItems(tc.items); !errors.Is(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
