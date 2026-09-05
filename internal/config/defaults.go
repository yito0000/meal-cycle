// Package config centralises meal-cycle's user-facing default values.
//
// docs/product.md #1 lists what is intended to be configurable. During MVP
// these are hard-coded constants returned by Defaults(). When the settings
// screen (S09) lands post-MVP, only this package changes — every caller
// keeps reading the same Config struct, now populated from a settings
// store rather than constants.
//
// Rule of thumb for future callers: never inline literal thresholds like
// "10 days" or "3 days" at the call site; take them from Config so they
// stay swappable in one place.
package config

// Config holds the meal-cycle runtime defaults.
//
// Keep this struct flat and JSON-serialisable; a future ConfigService may
// hand it to the frontend as-is.
type Config struct {
	// UI defaults ---------------------------------------------------------

	// DefaultVisibleSlots: slots that S01 has visible on first render.
	// Values match internal/domain/mealplan.Slot ("breakfast", "lunch",
	// "dinner"). Kept as []string here so this package stays free of a
	// domain import; the frontend can render each string 1:1 to a slot.
	DefaultVisibleSlots []string `json:"defaultVisibleSlots"`
	// WeeklyFillPattern: which cells the week-preset suggestion targets.
	// Free-form token for MVP (e.g. "mon-fri-dinner"); a richer type may
	// replace it when the setting UI lands.
	WeeklyFillPattern string `json:"weeklyFillPattern"`
	// ItemsPerSlotHint: soft target range for dish count per slot.
	// {min, max}. Not enforced — used only as a hint for suggestion
	// generation and (optionally) UI copy.
	ItemsPerSlotHint [2]int `json:"itemsPerSlotHint"`
	// MealPlanCellVisibleItems: how many dishes S01 renders inside one
	// (date, slot) cell before collapsing the rest into "他 N 件".
	MealPlanCellVisibleItems int `json:"mealPlanCellVisibleItems"`
	// RecipeListHighlightMs: milliseconds S02 keeps a just-saved recipe
	// row visually highlighted before clearing the search-param hint.
	// Used by S04 → S02 return flow (design.md D6).
	RecipeListHighlightMs int `json:"recipeListHighlightMs"`

	// Suggestion axis thresholds -----------------------------------------

	// RecencyThresholdDays: "久しぶりに" axis excludes recipes whose last
	// use is within this many days (accounts for leftover-cooking style
	// where the same recipe is eaten 2–3 days in a row).
	RecencyThresholdDays int `json:"recencyThresholdDays"`
	// LeftoverCopyPresets: N-day options offered by 作り置きコピー.
	LeftoverCopyPresets []int `json:"leftoverCopyPresets"`
	// WeeklyPresetRecipeCount: how many distinct recipes a week preset
	// should span. {min, max}.
	WeeklyPresetRecipeCount [2]int `json:"weeklyPresetRecipeCount"`

	// Expiry alerts -------------------------------------------------------

	// ExpiryAlertRedDays: 🔴 threshold (stock expiring within N days).
	ExpiryAlertRedDays int `json:"expiryAlertRedDays"`
	// ExpiryAlertYellowDays: 🟡 threshold (stock expiring within N days,
	// but past the red threshold).
	ExpiryAlertYellowDays int `json:"expiryAlertYellowDays"`
}

// Defaults returns the meal-cycle defaults tuned for the primary
// (leftover-cooking, dinner-focused) persona. See docs/product.md §1.
func Defaults() Config {
	return Config{
		DefaultVisibleSlots:      []string{"dinner"},
		WeeklyFillPattern:        "mon-fri-dinner",
		ItemsPerSlotHint:         [2]int{1, 3},
		MealPlanCellVisibleItems: 2,
		RecipeListHighlightMs:    2000,
		RecencyThresholdDays:     10,
		LeftoverCopyPresets:      []int{2, 3, 4},
		WeeklyPresetRecipeCount:  [2]int{2, 3},
		ExpiryAlertRedDays:       3,
		ExpiryAlertYellowDays:    7,
	}
}
