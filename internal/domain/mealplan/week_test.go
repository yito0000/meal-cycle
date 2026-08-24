package mealplan

import (
	"testing"
	"time"
)

func TestWeekStartOf(t *testing.T) {
	loc := time.FixedZone("JST", 9*60*60)

	cases := []struct {
		name string
		in   time.Time
		want time.Time
	}{
		{
			name: "monday at 00:00 returns the same day",
			in:   time.Date(2026, 8, 24, 0, 0, 0, 0, loc),
			want: time.Date(2026, 8, 24, 0, 0, 0, 0, loc),
		},
		{
			name: "sunday returns the previous monday (6 days earlier)",
			in:   time.Date(2026, 8, 30, 12, 34, 56, 0, loc),
			want: time.Date(2026, 8, 24, 0, 0, 0, 0, loc),
		},
		{
			name: "wednesday returns the monday 2 days earlier",
			in:   time.Date(2026, 8, 26, 15, 0, 0, 0, loc),
			want: time.Date(2026, 8, 24, 0, 0, 0, 0, loc),
		},
		{
			name: "time components are normalised to 00:00:00.000",
			in:   time.Date(2026, 8, 27, 23, 59, 59, 999_999_999, loc),
			want: time.Date(2026, 8, 24, 0, 0, 0, 0, loc),
		},
		{
			name: "location is preserved",
			in:   time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC),
			want: time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := WeekStartOf(tc.in)
			if !got.Equal(tc.want) {
				t.Errorf("WeekStartOf(%v) = %v, want %v", tc.in, got, tc.want)
			}
			if got.Location() != tc.in.Location() {
				t.Errorf("WeekStartOf location = %v, want %v", got.Location(), tc.in.Location())
			}
		})
	}
}
