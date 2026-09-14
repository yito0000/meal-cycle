package mealplan

import (
	"errors"
	"time"
)

// エラーコードは Wails bindings 経由でフロントに届き、i18n key にマップされる。
// 文言はここで持たず、`frontend/src/locales/*.json` の `mealplan.errors.*` に置く。
var (
	ErrInvalidDate         = errors.New("mealplan: invalid date")
	ErrInvalidMeal         = errors.New("mealplan: invalid meal")
	ErrInvalidPosition     = errors.New("mealplan: invalid position")
	ErrEntryNotFound       = errors.New("mealplan: entry not found")
	ErrReorderMismatch     = errors.New("mealplan: reorder mismatch")
	ErrInvalidCopyDays     = errors.New("mealplan: invalid copy days")
	ErrInvalidCopyMode     = errors.New("mealplan: invalid copy mode")
	ErrUnknownRecipe       = errors.New("mealplan: unknown recipe")
	ErrBatchTargetNotFound = errors.New("mealplan: batch target not found")
	ErrBatchMergeDenied    = errors.New("mealplan: batch merge denied")
	ErrInvalidPeriodWeeks  = errors.New("mealplan: invalid period weeks")
)

// AllowedPeriodWeeks は S01 で受け付ける期間長 (単位: 週) の集合。
// UI 側の期間切替トグル (1 / 2 / 4 週) と対応する。ここを増減するとフロントの
// PeriodNav / CellSizePreset マップも合わせて更新する。
var AllowedPeriodWeeks = []int{1, 2, 4}

// ValidatePeriodWeeks は AllowedPeriodWeeks のいずれかを許容する。
func ValidatePeriodWeeks(w int) error {
	for _, allowed := range AllowedPeriodWeeks {
		if w == allowed {
			return nil
		}
	}
	return ErrInvalidPeriodWeeks
}

// dateLayout はローカル日付を "YYYY-MM-DD" の厳密形式で扱う。time.RFC3339 の date-only 相当。
const dateLayout = "2006-01-02"

// ValidateDate は "YYYY-MM-DD" (厳密) を許容する。存在しない月日 (例: 2026-13-01) は
// time.Parse がエラーを返すのでそのまま弾ける。
func ValidateDate(s string) error {
	if s == "" {
		return ErrInvalidDate
	}
	if _, err := time.Parse(dateLayout, s); err != nil {
		return ErrInvalidDate
	}
	return nil
}

// ValidateMeal は breakfast|lunch|dinner のいずれかを許容する。大文字小文字の揺れは不許可。
func ValidateMeal(s string) error {
	switch Meal(s) {
	case MealBreakfast, MealLunch, MealDinner:
		return nil
	}
	return ErrInvalidMeal
}

func ValidatePosition(p int) error {
	if p < 0 {
		return ErrInvalidPosition
	}
	return nil
}
