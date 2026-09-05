package service

import (
	"context"

	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
)

// MealPlanService は S01 献立カレンダーのデータをフロントに公開する。
// 現状は固定 in-memory dataset を返す。SQLite 実装への差し替えは
// add-mealplan-assign-recipe で行う。
type MealPlanService struct{}

func NewMealPlanService() *MealPlanService {
	return &MealPlanService{}
}

// ListWeek は weekStart (月曜 "YYYY-MM-DD") が属する週の献立エントリを返す。
//
// 現状 weekStart は無視して固定 dataset を返す。パラメータをシグネチャに残して
// おき、後続 proposal がバインディング契約を変えずに repo lookup に差し替え
// られるようにする。dataset は Mon–Fri の夕食に加えて金曜に 4 品を並べており、
// フロントの「他 N 件」折り畳みを疎通確認できる。
func (s *MealPlanService) ListWeek(_ context.Context, weekStart string) ([]mealplan.Entry, error) {
	_ = weekStart
	return fixedWeekEntries(), nil
}

// 日付は 2026-08-24 (月) 起点。product.md のウォークスルーと一致させるため。
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
