package config

import "testing"

// 以下の値は docs/product.md §1 の source of truth。既定値を変えるときは
// このテストと product.md の両方を更新する。
func TestDefaults(t *testing.T) {
	c := Defaults()

	cases := []struct {
		name string
		got  any
		want any
	}{
		{"WeeklyFillPattern", c.WeeklyFillPattern, "mon-fri-dinner"},
		{"ItemsPerMealHint", c.ItemsPerMealHint, [2]int{1, 3}},
		{"MealPlanCellVisibleItems", c.MealPlanCellVisibleItems, 4},
		{"RecipeListHighlightMs", c.RecipeListHighlightMs, 2000},
		{"RecencyThresholdDays", c.RecencyThresholdDays, 10},
		{"WeeklyPresetRecipeCount", c.WeeklyPresetRecipeCount, [2]int{2, 3}},
		{"ExpiryAlertRedDays", c.ExpiryAlertRedDays, 3},
		{"ExpiryAlertYellowDays", c.ExpiryAlertYellowDays, 7},
		{"ShoppingDefaultPeriodDays", c.ShoppingDefaultPeriodDays, 7},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, tc.got, tc.want)
		}
	}

	if got, want := c.LeftoverCopyPresets, []int{2, 3, 4}; !equalInts(got, want) {
		t.Errorf("LeftoverCopyPresets: got %v, want %v", got, want)
	}

	if got, want := c.ShoppingPeriodDayPresets, []int{3, 4, 7, 14, 28}; !equalInts(got, want) {
		t.Errorf("ShoppingPeriodDayPresets: got %v, want %v", got, want)
	}
	if !containsInt(c.ShoppingPeriodDayPresets, c.ShoppingDefaultPeriodDays) {
		t.Errorf("ShoppingDefaultPeriodDays (%d) must be one of ShoppingPeriodDayPresets %v",
			c.ShoppingDefaultPeriodDays, c.ShoppingPeriodDayPresets)
	}
	// S01 の 1 / 2 / 4 週表示から期間を引き継ぐため (add-shopping-list design.md D9)。
	for _, days := range []int{7, 14, 28} {
		if !containsInt(c.ShoppingPeriodDayPresets, days) {
			t.Errorf("ShoppingPeriodDayPresets %v must contain %d", c.ShoppingPeriodDayPresets, days)
		}
	}

	if got, want := c.DefaultVisibleMeals, []string{"dinner"}; !equalStrings(got, want) {
		t.Errorf("DefaultVisibleMeals: got %v, want %v", got, want)
	}

	if c.ExpiryAlertRedDays >= c.ExpiryAlertYellowDays {
		t.Errorf("red (%d) must be stricter than yellow (%d)",
			c.ExpiryAlertRedDays, c.ExpiryAlertYellowDays)
	}
}

// MealPlanCellSizePresets の source of truth。design.md D4 の表を変えるときは
// このテストと product.md §S01 の両方を更新する。
func TestMealPlanCellSizePresets(t *testing.T) {
	presets := Defaults().MealPlanCellSizePresets

	t.Run("should return 9 presets covering 1/2/4 weeks x 1/2/3 meals", func(t *testing.T) {
		if len(presets) != 9 {
			t.Fatalf("preset count: got %d, want 9", len(presets))
		}
		seen := map[[2]int]bool{}
		for _, p := range presets {
			seen[[2]int{p.PeriodWeeks, p.VisibleMeals}] = true
		}
		for _, w := range []int{1, 2, 4} {
			for _, m := range []int{1, 2, 3} {
				if !seen[[2]int{w, m}] {
					t.Errorf("missing preset for periodWeeks=%d visibleMeals=%d", w, m)
				}
			}
		}
	})

	t.Run("should match D4 table for 1 week x 1 meal", func(t *testing.T) {
		p := findPreset(t, presets, 1, 1)
		wantPreset(t, p, 48, "text-base", 4)
	})

	t.Run("should match D4 table for 1 week x 3 meals", func(t *testing.T) {
		p := findPreset(t, presets, 1, 3)
		wantPreset(t, p, 32, "text-base", 3)
	})

	t.Run("should match D4 table for 4 weeks x 3 meals", func(t *testing.T) {
		p := findPreset(t, presets, 4, 3)
		wantPreset(t, p, 16, "text-xs", 2)
	})

	t.Run("should keep deprecated MealPlanCellVisibleItems equal to 1w x 1meal preset", func(t *testing.T) {
		p := findPreset(t, presets, 1, 1)
		if Defaults().MealPlanCellVisibleItems != p.VisibleItems {
			t.Errorf("MealPlanCellVisibleItems=%d, 1wx1meal.VisibleItems=%d — must match while deprecated",
				Defaults().MealPlanCellVisibleItems, p.VisibleItems)
		}
	})
}

func findPreset(t *testing.T, presets []CellSizePreset, weeks, meals int) CellSizePreset {
	t.Helper()
	for _, p := range presets {
		if p.PeriodWeeks == weeks && p.VisibleMeals == meals {
			return p
		}
	}
	t.Fatalf("preset not found: periodWeeks=%d visibleMeals=%d", weeks, meals)
	return CellSizePreset{}
}

func wantPreset(t *testing.T, p CellSizePreset, minH int, font string, items int) {
	t.Helper()
	if p.MinHeightRem != minH {
		t.Errorf("MinHeightRem: got %d, want %d", p.MinHeightRem, minH)
	}
	if p.FontSizeClass != font {
		t.Errorf("FontSizeClass: got %q, want %q", p.FontSizeClass, font)
	}
	if p.VisibleItems != items {
		t.Errorf("VisibleItems: got %d, want %d", p.VisibleItems, items)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
