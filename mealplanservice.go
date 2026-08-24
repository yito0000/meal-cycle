package main

import (
	"context"

	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
)

// MealPlanService exposes the meal calendar (S01) data to the frontend.
//
// This proposal (add-mealplan-week-view) keeps the implementation as a
// fixed in-memory dataset so the UI, the Wails binding and the domain
// types can be locked in before repository-backed storage lands with
// add-mealplan-assign-recipe.
type MealPlanService struct{}

// NewMealPlanService constructs a MealPlanService.
func NewMealPlanService() *MealPlanService {
	return &MealPlanService{}
}

// ListWeek returns every meal-plan entry that belongs to the week that
// starts on weekStart (a "YYYY-MM-DD" Monday). Until the repository
// lands, the response is a fixed set that covers Mon–Fri dinners plus
// one Friday overflow slot (4 dishes) so the frontend can exercise the
// "他 N 件" collapse.
//
// weekStart is currently ignored — every call returns the same fixed
// dataset. The parameter is kept in the signature so add-mealplan-
// assign-recipe can swap the body for a repo lookup without changing
// the Wails binding surface.
func (s *MealPlanService) ListWeek(_ context.Context, weekStart string) ([]mealplan.Entry, error) {
	_ = weekStart // reserved for the repository swap in the next proposal
	return fixedWeekEntries(), nil
}

// fixedWeekEntries returns the placeholder dataset used by ListWeek.
// Dates are anchored to the ISO week starting 2026-08-24 (Monday) so
// the fixture matches the wireframe walkthroughs in docs/product.md.
func fixedWeekEntries() []mealplan.Entry {
	return []mealplan.Entry{
		{Date: "2026-08-24", Slot: mealplan.SlotDinner, Position: 0, RecipeName: "肉じゃが"},
		{Date: "2026-08-24", Slot: mealplan.SlotDinner, Position: 1, RecipeName: "みそ汁"},

		{Date: "2026-08-25", Slot: mealplan.SlotDinner, Position: 0, RecipeName: "肉じゃが"},

		{Date: "2026-08-26", Slot: mealplan.SlotDinner, Position: 0, RecipeName: "麻婆豆腐"},
		{Date: "2026-08-26", Slot: mealplan.SlotDinner, Position: 1, RecipeName: "中華スープ"},

		{Date: "2026-08-27", Slot: mealplan.SlotDinner, Position: 0, RecipeName: "鮭のホイル焼き"},
		{Date: "2026-08-27", Slot: mealplan.SlotLunch, Position: 0, RecipeName: "焼きうどん"},
		{Date: "2026-08-27", Slot: mealplan.SlotBreakfast, Position: 0, RecipeName: "トースト"},

		{Date: "2026-08-28", Slot: mealplan.SlotDinner, Position: 0, RecipeName: "肉じゃが"},
		{Date: "2026-08-28", Slot: mealplan.SlotDinner, Position: 1, RecipeName: "ほうれん草の胡麻和え"},
		{Date: "2026-08-28", Slot: mealplan.SlotDinner, Position: 2, RecipeName: "みそ汁"},
		{Date: "2026-08-28", Slot: mealplan.SlotDinner, Position: 3, RecipeName: "ごはん"},
	}
}
