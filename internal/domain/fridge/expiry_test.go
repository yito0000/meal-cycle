package fridge

import (
	"testing"
	"time"
)

func TestEvaluate(t *testing.T) {
	loc := time.FixedZone("JST", 9*60*60)
	today := time.Date(2026, 9, 23, 0, 0, 0, 0, loc)
	const red, yellow = 3, 7

	cases := []struct {
		name       string
		expiresAt  string
		today      time.Time
		wantStatus ExpiryStatus
		wantDays   *int
	}{
		{name: "empty expiry is none with nil days", expiresAt: "", today: today, wantStatus: ExpiryNone},
		{name: "unparseable expiry is none with nil days", expiresAt: "2026/09/30", today: today, wantStatus: ExpiryNone},
		{name: "8 days left is fresh", expiresAt: "2026-10-01", today: today, wantStatus: ExpiryFresh, wantDays: ptr(8)},
		{name: "7 days left (yellow boundary) is soon", expiresAt: "2026-09-30", today: today, wantStatus: ExpirySoon, wantDays: ptr(7)},
		{name: "4 days left is soon", expiresAt: "2026-09-27", today: today, wantStatus: ExpirySoon, wantDays: ptr(4)},
		{name: "3 days left (red boundary) is urgent", expiresAt: "2026-09-26", today: today, wantStatus: ExpiryUrgent, wantDays: ptr(3)},
		{name: "expiring today is urgent with 0 days", expiresAt: "2026-09-23", today: today, wantStatus: ExpiryUrgent, wantDays: ptr(0)},
		{name: "expired 2 days ago is expired with -2 days", expiresAt: "2026-09-21", today: today, wantStatus: ExpiryExpired, wantDays: ptr(-2)},
		{name: "expired 1 day ago is expired", expiresAt: "2026-09-22", today: today, wantStatus: ExpiryExpired, wantDays: ptr(-1)},
		{
			name:       "time of day does not affect days left",
			expiresAt:  "2026-09-24",
			today:      time.Date(2026, 9, 23, 23, 59, 59, 0, loc),
			wantStatus: ExpiryUrgent,
			wantDays:   ptr(1),
		},
		{
			name:       "early morning in JST uses the local date",
			expiresAt:  "2026-09-24",
			today:      time.Date(2026, 9, 23, 1, 0, 0, 0, loc),
			wantStatus: ExpiryUrgent,
			wantDays:   ptr(1),
		},
		{
			name:       "days left across month end",
			expiresAt:  "2026-10-02",
			today:      time.Date(2026, 9, 28, 0, 0, 0, 0, loc),
			wantStatus: ExpirySoon,
			wantDays:   ptr(4),
		},
		{
			name:       "days left across year end",
			expiresAt:  "2027-01-05",
			today:      time.Date(2026, 12, 30, 0, 0, 0, 0, loc),
			wantStatus: ExpirySoon,
			wantDays:   ptr(6),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotStatus, gotDays := Evaluate(tc.expiresAt, tc.today, red, yellow)
			if gotStatus != tc.wantStatus {
				t.Errorf("status: want %q, got %q", tc.wantStatus, gotStatus)
			}
			switch {
			case tc.wantDays == nil && gotDays != nil:
				t.Errorf("days: want nil, got %d", *gotDays)
			case tc.wantDays != nil && gotDays == nil:
				t.Errorf("days: want %d, got nil", *tc.wantDays)
			case tc.wantDays != nil && *gotDays != *tc.wantDays:
				t.Errorf("days: want %d, got %d", *tc.wantDays, *gotDays)
			}
		})
	}
}

func ptr(n int) *int { return &n }
