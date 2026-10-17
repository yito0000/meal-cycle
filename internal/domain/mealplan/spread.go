package mealplan

import "time"

// SpreadTarget は「作り置きを広げる」で選んだ 1 セル。
type SpreadTarget struct {
	Date string `json:"date"`
	Meal Meal   `json:"meal"`
}

// SpreadResult は「作り置きを広げる」の取り消しに使う結果。Removed は上書きで消した行。
type SpreadResult struct {
	AddedEntryIDs []string       `json:"addedEntryIds"`
	Removed       []RemovedEntry `json:"removed"`
}

// ValidateSpreadTargets は起点セルから広げる先として targets が使えるかを確かめる。
// 作り置きは起点より後のセルに伸ばす操作なので、起点日より前の日付と、起点日の起点以前の時間帯は受け付けない
// (revise-mealplan-spread-safety design.md D4)。
func ValidateSpreadTargets(sourceDate string, sourceMeal Meal, targets []SpreadTarget, maxOffsetDays int) error {
	if err := ValidateDate(sourceDate); err != nil {
		return err
	}
	if err := ValidateMeal(string(sourceMeal)); err != nil {
		return err
	}
	if len(targets) == 0 {
		return ErrInvalidSpreadTarget
	}
	start, _ := time.Parse(dateLayout, sourceDate)
	last := start.AddDate(0, 0, maxOffsetDays)
	seen := make(map[SpreadTarget]bool, len(targets))
	for _, tg := range targets {
		if err := ValidateDate(tg.Date); err != nil {
			return err
		}
		if err := ValidateMeal(string(tg.Meal)); err != nil {
			return err
		}
		if tg.Date == sourceDate && MealRank(tg.Meal) <= MealRank(sourceMeal) {
			return ErrInvalidSpreadTarget
		}
		d, _ := time.Parse(dateLayout, tg.Date)
		if d.Before(start) || d.After(last) {
			return ErrInvalidSpreadTarget
		}
		if seen[tg] {
			return ErrInvalidSpreadTarget
		}
		seen[tg] = true
	}
	return nil
}
