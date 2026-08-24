// Package mealplan is the domain layer for the meal calendar (S01).
//
// A MealPlan is stored as "one row = one dish": the same (date, slot)
// can carry 0..N Entry rows so a single meal can hold main + side + soup.
// See docs/design.md §4 (MealPlan) for the rationale.
//
// This package is intentionally framework-free; the Wails service and
// the SQLite repository (added by later proposals) sit above it.
package mealplan

// Slot names a slot within a day. Values match the SQLite column values
// used by future migrations, so keep the underlying strings stable.
type Slot string

const (
	SlotBreakfast Slot = "breakfast"
	SlotLunch     Slot = "lunch"
	SlotDinner    Slot = "dinner"
)

// Entry is one dish assigned to (Date, Slot) at a given Position.
//
// Date is "YYYY-MM-DD" (local calendar date, no timezone). Position is
// 0-based and unique within a (Date, Slot) — it controls the vertical
// stacking order in the S01 cell (main → side → soup).
//
// RecipeName is the current source of truth for display; RecipeID will
// be added when the recipe capability lands and repository-backed
// entries replace the fixed values used by add-mealplan-week-view.
type Entry struct {
	Date       string `json:"date"`
	Slot       Slot   `json:"slot"`
	Position   int    `json:"position"`
	RecipeName string `json:"recipeName"`
}
