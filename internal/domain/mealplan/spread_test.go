package mealplan

import (
	"errors"
	"testing"
)

func TestValidateSpreadTargets(t *testing.T) {
	const src = "2026-08-24"
	cases := []struct {
		name    string
		targets []SpreadTarget
		want    error
	}{
		{
			name: "same-meal and other-meal cells from the next day up to 3 days later pass",
			targets: []SpreadTarget{
				{Date: "2026-08-25", Meal: MealDinner},
				{Date: "2026-08-26", Meal: MealLunch},
				{Date: "2026-08-27", Meal: MealBreakfast},
				{Date: "2026-08-27", Meal: MealDinner},
			},
			want: nil,
		},
		{
			name:    "other-meal cell on the source date passes",
			targets: []SpreadTarget{{Date: src, Meal: MealLunch}},
			want:    nil,
		},
		{
			name:    "empty targets is ErrInvalidSpreadTarget",
			targets: []SpreadTarget{},
			want:    ErrInvalidSpreadTarget,
		},
		{
			name:    "source cell itself is ErrInvalidSpreadTarget",
			targets: []SpreadTarget{{Date: "2026-08-25", Meal: MealDinner}, {Date: src, Meal: MealDinner}},
			want:    ErrInvalidSpreadTarget,
		},
		{
			name:    "date before the source date is ErrInvalidSpreadTarget",
			targets: []SpreadTarget{{Date: "2026-08-23", Meal: MealDinner}},
			want:    ErrInvalidSpreadTarget,
		},
		{
			name:    "date after source date + maxOffsetDays is ErrInvalidSpreadTarget",
			targets: []SpreadTarget{{Date: "2026-08-28", Meal: MealDinner}},
			want:    ErrInvalidSpreadTarget,
		},
		{
			name:    "duplicated cell is ErrInvalidSpreadTarget",
			targets: []SpreadTarget{{Date: "2026-08-25", Meal: MealDinner}, {Date: "2026-08-25", Meal: MealDinner}},
			want:    ErrInvalidSpreadTarget,
		},
		{
			name:    "malformed date is ErrInvalidDate",
			targets: []SpreadTarget{{Date: "2026/08/25", Meal: MealDinner}},
			want:    ErrInvalidDate,
		},
		{
			name:    "unknown meal is ErrInvalidMeal",
			targets: []SpreadTarget{{Date: "2026-08-25", Meal: Meal("supper")}},
			want:    ErrInvalidMeal,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSpreadTargets(src, MealDinner, tc.targets, 3)
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}
