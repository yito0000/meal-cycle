package mealplan

import "time"

// WeekStartOf は t が属する週の月曜 00:00 (ローカル) を返す。
//
// 月曜起点にするのは、想定ペルソナが「日曜夜に来週の献立を組む」ため
// (product.md §1)。返り値は Location を保持し、時分秒ナノ秒はゼロなので
// 日付比較に使える。
func WeekStartOf(t time.Time) time.Time {
	// time.Weekday は Sun=0, Mon=1, .., Sat=6。月曜までの距離は (weekday + 6) % 7
	// (Sun → 6, Mon → 0)。
	offset := (int(t.Weekday()) + 6) % 7
	monday := t.AddDate(0, 0, -offset)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, t.Location())
}
