package config

import "testing"

// The values below are the source of truth for docs/product.md §1.
// If a default changes, update both this test and the doc.
func TestDefaults(t *testing.T) {
	c := Defaults()

	cases := []struct {
		name string
		got  any
		want any
	}{
		{"WeeklyFillPattern", c.WeeklyFillPattern, "mon-fri-dinner"},
		{"ItemsPerSlotHint", c.ItemsPerSlotHint, [2]int{1, 3}},
		{"MealPlanCellVisibleItems", c.MealPlanCellVisibleItems, 2},
		{"RecipeListHighlightMs", c.RecipeListHighlightMs, 2000},
		{"RecencyThresholdDays", c.RecencyThresholdDays, 10},
		{"WeeklyPresetRecipeCount", c.WeeklyPresetRecipeCount, [2]int{2, 3}},
		{"ExpiryAlertRedDays", c.ExpiryAlertRedDays, 3},
		{"ExpiryAlertYellowDays", c.ExpiryAlertYellowDays, 7},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, tc.got, tc.want)
		}
	}

	if got, want := c.LeftoverCopyPresets, []int{2, 3, 4}; !equalInts(got, want) {
		t.Errorf("LeftoverCopyPresets: got %v, want %v", got, want)
	}

	if got, want := c.DefaultVisibleSlots, []string{"dinner"}; !equalStrings(got, want) {
		t.Errorf("DefaultVisibleSlots: got %v, want %v", got, want)
	}

	if c.ExpiryAlertRedDays >= c.ExpiryAlertYellowDays {
		t.Errorf("red (%d) must be stricter than yellow (%d)",
			c.ExpiryAlertRedDays, c.ExpiryAlertYellowDays)
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
