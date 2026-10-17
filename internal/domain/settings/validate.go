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
)

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
