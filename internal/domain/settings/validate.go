package settings

import (
	"slices"
	"unicode/utf8"
)

// 実装上の上限で、利用者が変える値ではないので defaults.go には置かない (design.md D5)。
const (
	ExpiryAlertRedDaysMin    = 0
	ExpiryAlertYellowDaysMax = 60

	LeftoverPresetsMin = 1
	LeftoverPresetsMax = 5
	LeftoverDaysMin    = 2
	// 「作り置きを広げる」の格子の行数と「他の作り置きにまとめる」の探索範囲がこの値で広がるので、1 週間で止める。
	LeftoverDaysMax = 7

	ShoppingPresetsMax = 8
	ShoppingDaysMin    = 1
	ShoppingDaysMax    = 28

	MeasuringUnitsMax    = 10
	UnquantifiedWordsMax = 20
	WordRunesMin         = 1
	WordRunesMax         = 20

	ImportServingsMin = 1
	ImportServingsMax = 99

	RecencyDaysMin = 1
	// 週単位の献立で、2 か月より前の履歴を「久しぶり」の基準に使う場面が見当たらないので止める (add-mealplan-suggest-by-recency design.md D5)。
	RecencyDaysMax = 60

	WeekdayMin = 1
	WeekdayMax = 7
	// 1 週間を超えて同じ品を続けても、対象の週の中には収まらないので止める。
	WeeklyRunDaysMin = 1
	WeeklyRunDaysMax = 7
)

// 週の候補で埋める時間帯。internal/domain/mealplan.Meal と同じ文字列 (settings が mealplan を import しないため写す)。
var weeklyFillMeals = []string{"breakfast", "lunch", "dinner"}

// S01 の 1 / 2 / 4 週の期間を S07 に引き継ぐので、買い物リストの候補から外せない。
var ShoppingRequiredDays = []int{7, 14, 28}

var languages = []string{"ja", "en"}

// Validate は key ごとの検証を順に呼び、最初のエラーを返す。Normalize 済みの値を渡す。
func (v Values) Validate() error {
	checks := []func() error{
		func() error { return ValidateLanguage(v.Language) },
		func() error { return ValidateExpiryAlert(v.ExpiryAlertRedDays, v.ExpiryAlertYellowDays) },
		func() error { return ValidateLeftoverCopyPresets(v.LeftoverCopyPresets) },
		func() error { return ValidateShoppingPeriod(v.ShoppingPeriodDayPresets, v.ShoppingDefaultPeriodDays) },
		func() error { return ValidateMeasuringUnits(v.MeasuringUnits) },
		func() error { return ValidateUnquantifiedWords(v.UnquantifiedWords) },
		func() error { return ValidateImportServings(v.RecipeImportDefaultServings) },
		func() error { return ValidateRecencyThreshold(v.RecencyThresholdDays) },
		func() error { return ValidateWeeklyFill(v.WeeklyFillDays, v.WeeklyFillMeal) },
		func() error { return ValidateWeeklyPresetMaxRunDays(v.WeeklyPresetMaxRunDays) },
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

func ValidateLanguage(lang string) error {
	if !slices.Contains(languages, lang) {
		return ErrInvalidLanguage
	}
	return nil
}

func ValidateExpiryAlert(red, yellow int) error {
	if red < ExpiryAlertRedDaysMin {
		return ErrRedDaysOutOfRange
	}
	if yellow > ExpiryAlertYellowDaysMax {
		return ErrYellowDaysOutOfRange
	}
	if red >= yellow {
		return ErrRedNotBelowYellow
	}
	return nil
}

func ValidateLeftoverCopyPresets(days []int) error {
	if len(days) < LeftoverPresetsMin || len(days) > LeftoverPresetsMax {
		return ErrLeftoverPresetsCount
	}
	for _, d := range days {
		if d < LeftoverDaysMin || d > LeftoverDaysMax {
			return ErrLeftoverPresetOutOfRange
		}
	}
	if hasDuplicate(days) {
		return ErrLeftoverPresetDuplicate
	}
	return nil
}

func ValidateShoppingPeriod(days []int, defaultDays int) error {
	if len(days) > ShoppingPresetsMax {
		return ErrShoppingPresetsCount
	}
	for _, d := range days {
		if d < ShoppingDaysMin || d > ShoppingDaysMax {
			return ErrShoppingPresetOutOfRange
		}
	}
	if hasDuplicate(days) {
		return ErrShoppingPresetDuplicate
	}
	for _, d := range ShoppingRequiredDays {
		if !slices.Contains(days, d) {
			return ErrShoppingPresetRequiredMissing
		}
	}
	if !slices.Contains(days, defaultDays) {
		return ErrShoppingDefaultNotInPresets
	}
	return nil
}

func ValidateMeasuringUnits(units []string) error {
	return validateWords(units, MeasuringUnitsMax,
		ErrMeasuringUnitsCount, ErrMeasuringUnitLength, ErrMeasuringUnitDuplicate)
}

func ValidateUnquantifiedWords(words []string) error {
	return validateWords(words, UnquantifiedWordsMax,
		ErrUnquantifiedWordsCount, ErrUnquantifiedWordLength, ErrUnquantifiedWordDuplicate)
}

func ValidateImportServings(servings int) error {
	if servings < ImportServingsMin || servings > ImportServingsMax {
		return ErrImportServingsOutOfRange
	}
	return nil
}

func ValidateRecencyThreshold(days int) error {
	if days < RecencyDaysMin || days > RecencyDaysMax {
		return ErrRecencyDaysOutOfRange
	}
	return nil
}

func ValidateWeeklyFill(days []int, meal string) error {
	if len(days) == 0 {
		return ErrWeeklyFillDaysEmpty
	}
	for _, d := range days {
		if d < WeekdayMin || d > WeekdayMax {
			return ErrWeeklyFillDayOutOfRange
		}
	}
	if hasDuplicate(days) {
		return ErrWeeklyFillDayDuplicate
	}
	if !slices.Contains(weeklyFillMeals, meal) {
		return ErrWeeklyFillInvalidMeal
	}
	return nil
}

func ValidateWeeklyPresetMaxRunDays(days int) error {
	if days < WeeklyRunDaysMin || days > WeeklyRunDaysMax {
		return ErrWeeklyRunDaysOutOfRange
	}
	return nil
}

func validateWords(words []string, max int, errCount, errLength, errDuplicate error) error {
	if len(words) > max {
		return errCount
	}
	for _, w := range words {
		n := utf8.RuneCountInString(w)
		if n < WordRunesMin || n > WordRunesMax {
			return errLength
		}
	}
	if hasDuplicate(words) {
		return errDuplicate
	}
	return nil
}

func hasDuplicate[T comparable](xs []T) bool {
	seen := make(map[T]struct{}, len(xs))
	for _, x := range xs {
		if _, ok := seen[x]; ok {
			return true
		}
		seen[x] = struct{}{}
	}
	return false
}
