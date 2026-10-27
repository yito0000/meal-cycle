// Package settings は S09 設定画面で変えられる値と、その正規化と検証を扱う。
package settings

import (
	"slices"
	"strings"

	"github.com/yito0000/meal-cycle/internal/config"
)

// Values は S09 で変えられる項目だけを持つ。Config 全体を返すとセルサイズの表などまでフロントに見えるため分ける
// (add-settings-screen design.md D4)。
type Values struct {
	Language                    string   `json:"language"`
	ExpiryAlertRedDays          int      `json:"expiryAlertRedDays"`
	ExpiryAlertYellowDays       int      `json:"expiryAlertYellowDays"`
	LeftoverCopyPresets         []int    `json:"leftoverCopyPresets"`
	ShoppingPeriodDayPresets    []int    `json:"shoppingPeriodDayPresets"`
	ShoppingDefaultPeriodDays   int      `json:"shoppingDefaultPeriodDays"`
	MeasuringUnits              []string `json:"measuringUnits"`
	UnquantifiedWords           []string `json:"unquantifiedWords"`
	RecipeImportDefaultServings int      `json:"recipeImportDefaultServings"`
	RecencyThresholdDays        int      `json:"recencyThresholdDays"`
	WeeklyFillDays              []int    `json:"weeklyFillDays"`
	WeeklyFillMeal              string   `json:"weeklyFillMeal"`
	WeeklyPresetMaxRunDays      int      `json:"weeklyPresetMaxRunDays"`
}

func FromConfig(c config.Config) Values {
	return Values{
		Language:                    c.Language,
		ExpiryAlertRedDays:          c.ExpiryAlertRedDays,
		ExpiryAlertYellowDays:       c.ExpiryAlertYellowDays,
		LeftoverCopyPresets:         cloneOrEmpty(c.LeftoverCopyPresets),
		ShoppingPeriodDayPresets:    cloneOrEmpty(c.ShoppingPeriodDayPresets),
		ShoppingDefaultPeriodDays:   c.ShoppingDefaultPeriodDays,
		MeasuringUnits:              cloneOrEmpty(c.MeasuringUnits),
		UnquantifiedWords:           cloneOrEmpty(c.UnquantifiedWords),
		RecipeImportDefaultServings: c.RecipeImportDefaultServings,
		RecencyThresholdDays:        c.RecencyThresholdDays,
		WeeklyFillDays:              cloneOrEmpty(c.WeeklyFillDays),
		WeeklyFillMeal:              c.WeeklyFillMeal,
		WeeklyPresetMaxRunDays:      c.WeeklyPresetMaxRunDays,
	}
}

// ApplyTo は base のうち S09 で変えられる項目だけを v で上書きした Config を返す。
func (v Values) ApplyTo(base config.Config) config.Config {
	base.Language = v.Language
	base.ExpiryAlertRedDays = v.ExpiryAlertRedDays
	base.ExpiryAlertYellowDays = v.ExpiryAlertYellowDays
	base.LeftoverCopyPresets = cloneOrEmpty(v.LeftoverCopyPresets)
	base.ShoppingPeriodDayPresets = cloneOrEmpty(v.ShoppingPeriodDayPresets)
	base.ShoppingDefaultPeriodDays = v.ShoppingDefaultPeriodDays
	base.MeasuringUnits = cloneOrEmpty(v.MeasuringUnits)
	base.UnquantifiedWords = cloneOrEmpty(v.UnquantifiedWords)
	base.RecipeImportDefaultServings = v.RecipeImportDefaultServings
	base.RecencyThresholdDays = v.RecencyThresholdDays
	base.WeeklyFillDays = cloneOrEmpty(v.WeeklyFillDays)
	base.WeeklyFillMeal = v.WeeklyFillMeal
	base.WeeklyPresetMaxRunDays = v.WeeklyPresetMaxRunDays
	return base
}

// Normalize は並べ替えや前後の空白だけが違う値を同じ値にそろえる (design.md D10)。
func (v Values) Normalize() Values {
	v.LeftoverCopyPresets = sortedCopy(v.LeftoverCopyPresets)
	v.ShoppingPeriodDayPresets = sortedCopy(v.ShoppingPeriodDayPresets)
	v.WeeklyFillDays = sortedCopy(v.WeeklyFillDays)
	v.MeasuringUnits = trimmedCopy(v.MeasuringUnits)
	v.UnquantifiedWords = trimmedCopy(v.UnquantifiedWords)
	return v
}

// nil はフロントに null で届き配列として扱えないので、空でも非 nil にそろえる。
func cloneOrEmpty[T any](xs []T) []T {
	out := make([]T, len(xs))
	copy(out, xs)
	return out
}

func sortedCopy(xs []int) []int {
	out := cloneOrEmpty(xs)
	slices.Sort(out)
	return out
}

func trimmedCopy(xs []string) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = strings.TrimSpace(x)
	}
	return out
}
