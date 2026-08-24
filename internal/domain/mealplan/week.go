package mealplan

import "time"

// WeekStartOf returns the Monday 00:00 (local) of the ISO week that
// contains t. The primary persona plans "next week" on Sunday night,
// so Monday-start matches product.md §1's narrative and lines up with
// ISO 8601's week numbering.
//
// The returned time has t's Location preserved; hour/minute/second/
// nanosecond are zeroed so callers can compare dates safely.
func WeekStartOf(t time.Time) time.Time {
	// time.Weekday: Sunday=0, Monday=1, ..., Saturday=6.
	// Distance back to Monday: (weekday + 6) % 7. Sunday → 6, Monday → 0.
	offset := (int(t.Weekday()) + 6) % 7
	monday := t.AddDate(0, 0, -offset)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, t.Location())
}
