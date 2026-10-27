package settings

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/yito0000/meal-cycle/internal/config"
)

func TestFromConfigApplyTo(t *testing.T) {
	want := config.Defaults()
	got := FromConfig(want).ApplyTo(config.Defaults())
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip mismatch:\n got  %#v\n want %#v", got, want)
	}
}

func TestNormalize(t *testing.T) {
	cases := []struct {
		name  string
		input Values
		check func(t *testing.T, got Values)
	}{
		{
			name:  "trims spaces around units",
			input: Values{MeasuringUnits: []string{" 大さじ ", "\t合"}, UnquantifiedWords: []string{" 少々"}},
			check: func(t *testing.T, got Values) {
				if want := []string{"大さじ", "合"}; !reflect.DeepEqual(got.MeasuringUnits, want) {
					t.Errorf("MeasuringUnits = %v, want %v", got.MeasuringUnits, want)
				}
				if want := []string{"少々"}; !reflect.DeepEqual(got.UnquantifiedWords, want) {
					t.Errorf("UnquantifiedWords = %v, want %v", got.UnquantifiedWords, want)
				}
			},
		},
		{
			name:  "sorts day presets ascending",
			input: Values{LeftoverCopyPresets: []int{4, 2, 3}, ShoppingPeriodDayPresets: []int{28, 7, 3, 14}},
			check: func(t *testing.T, got Values) {
				if want := []int{2, 3, 4}; !reflect.DeepEqual(got.LeftoverCopyPresets, want) {
					t.Errorf("LeftoverCopyPresets = %v, want %v", got.LeftoverCopyPresets, want)
				}
				if want := []int{3, 7, 14, 28}; !reflect.DeepEqual(got.ShoppingPeriodDayPresets, want) {
					t.Errorf("ShoppingPeriodDayPresets = %v, want %v", got.ShoppingPeriodDayPresets, want)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.check(t, tc.input.Normalize())
		})
	}
}

func TestValidate(t *testing.T) {
	words := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = strings.Repeat("あ", i+1)
		}
		return out
	}

	cases := []struct {
		name   string
		modify func(v *Values)
		want   error
	}{
		{"defaults pass", func(*Values) {}, nil},
		{"unknown language", func(v *Values) { v.Language = "fr" }, ErrInvalidLanguage},
		{"red days -1", func(v *Values) { v.ExpiryAlertRedDays = -1 }, ErrRedDaysOutOfRange},
		{"yellow days 61", func(v *Values) { v.ExpiryAlertYellowDays = 61 }, ErrYellowDaysOutOfRange},
		{"red 7 yellow 7", func(v *Values) { v.ExpiryAlertRedDays, v.ExpiryAlertYellowDays = 7, 7 }, ErrRedNotBelowYellow},
		{"red 0 yellow 1", func(v *Values) { v.ExpiryAlertRedDays, v.ExpiryAlertYellowDays = 0, 1 }, nil},
		{"empty leftover presets", func(v *Values) { v.LeftoverCopyPresets = []int{} }, ErrLeftoverPresetsCount},
		{"six leftover presets", func(v *Values) { v.LeftoverCopyPresets = []int{2, 3, 4, 5, 6, 7} }, ErrLeftoverPresetsCount},
		{"leftover preset 1", func(v *Values) { v.LeftoverCopyPresets = []int{1, 3} }, ErrLeftoverPresetOutOfRange},
		{"leftover preset 8", func(v *Values) { v.LeftoverCopyPresets = []int{3, 8} }, ErrLeftoverPresetOutOfRange},
		{"duplicate leftover preset", func(v *Values) { v.LeftoverCopyPresets = []int{3, 3} }, ErrLeftoverPresetDuplicate},
		{"shopping presets without 14", func(v *Values) { v.ShoppingPeriodDayPresets = []int{3, 7, 28} },
			ErrShoppingPresetRequiredMissing},
		{"shopping preset 29", func(v *Values) { v.ShoppingPeriodDayPresets = []int{7, 14, 28, 29} },
			ErrShoppingPresetOutOfRange},
		{"nine shopping presets", func(v *Values) {
			v.ShoppingPeriodDayPresets = []int{1, 2, 3, 4, 5, 6, 7, 14, 28}
		}, ErrShoppingPresetsCount},
		{"shopping default 5 not in presets", func(v *Values) { v.ShoppingDefaultPeriodDays = 5 },
			ErrShoppingDefaultNotInPresets},
		{"measuring unit of 21 runes", func(v *Values) { v.MeasuringUnits = []string{strings.Repeat("あ", 21)} },
			ErrMeasuringUnitLength},
		{"empty measuring unit", func(v *Values) { v.MeasuringUnits = []string{""} }, ErrMeasuringUnitLength},
		{"duplicate measuring unit", func(v *Values) { v.MeasuringUnits = []string{"大さじ", "大さじ"} },
			ErrMeasuringUnitDuplicate},
		{"eleven measuring units", func(v *Values) { v.MeasuringUnits = words(11) }, ErrMeasuringUnitsCount},
		{"no measuring units", func(v *Values) { v.MeasuringUnits = []string{} }, nil},
		{"21 unquantified words", func(v *Values) { v.UnquantifiedWords = words(21) }, ErrUnquantifiedWordsCount},
		{"servings 0", func(v *Values) { v.RecipeImportDefaultServings = 0 }, ErrImportServingsOutOfRange},
		{"servings 100", func(v *Values) { v.RecipeImportDefaultServings = 100 }, ErrImportServingsOutOfRange},
		{"recency days 1", func(v *Values) { v.RecencyThresholdDays = 1 }, nil},
		{"recency days 60", func(v *Values) { v.RecencyThresholdDays = 60 }, nil},
		{"recency days 0", func(v *Values) { v.RecencyThresholdDays = 0 }, ErrRecencyDaysOutOfRange},
		{"recency days 61", func(v *Values) { v.RecencyThresholdDays = 61 }, ErrRecencyDaysOutOfRange},
		{"empty weekly fill days", func(v *Values) { v.WeeklyFillDays = []int{} }, ErrWeeklyFillDaysEmpty},
		{"weekly fill day 0", func(v *Values) { v.WeeklyFillDays = []int{0, 1} }, ErrWeeklyFillDayOutOfRange},
		{"weekly fill day 8", func(v *Values) { v.WeeklyFillDays = []int{1, 8} }, ErrWeeklyFillDayOutOfRange},
		{"duplicate weekly fill day", func(v *Values) { v.WeeklyFillDays = []int{1, 1} }, ErrWeeklyFillDayDuplicate},
		{"unknown weekly fill meal", func(v *Values) { v.WeeklyFillMeal = "brunch" }, ErrWeeklyFillInvalidMeal},
		{"weekly run days 0", func(v *Values) { v.WeeklyPresetMaxRunDays = 0 }, ErrWeeklyRunDaysOutOfRange},
		{"weekly run days 8", func(v *Values) { v.WeeklyPresetMaxRunDays = 8 }, ErrWeeklyRunDaysOutOfRange},
		{"weekly run days 1", func(v *Values) { v.WeeklyPresetMaxRunDays = 1 }, nil},
		{"weekly run days 7", func(v *Values) { v.WeeklyPresetMaxRunDays = 7 }, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := FromConfig(config.Defaults())
			tc.modify(&v)
			if got := v.Validate(); !errors.Is(got, tc.want) {
				t.Errorf("Validate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCurrentConstraints(t *testing.T) {
	cases := []struct {
		name  string
		check func(t *testing.T, c Constraints)
	}{
		{"required shopping days are 7, 14 and 28", func(t *testing.T, c Constraints) {
			if want := []int{7, 14, 28}; !reflect.DeepEqual(c.ShoppingRequiredDays, want) {
				t.Errorf("required = %v, want %v", c.ShoppingRequiredDays, want)
			}
		}},
		{"fallback days is one of the required days", func(t *testing.T, c Constraints) {
			if !slices.Contains(c.ShoppingRequiredDays, c.ShoppingFallbackDays) {
				t.Errorf("fallback %d not in %v", c.ShoppingFallbackDays, c.ShoppingRequiredDays)
			}
		}},
		{"mutating the returned slice keeps ShoppingRequiredDays", func(t *testing.T, c Constraints) {
			c.ShoppingRequiredDays[0] = 99
			if ShoppingRequiredDays[0] != 7 {
				t.Errorf("ShoppingRequiredDays = %v, want unchanged", ShoppingRequiredDays)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { tc.check(t, CurrentConstraints()) })
	}
}
