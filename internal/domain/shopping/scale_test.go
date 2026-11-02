package shopping

import "testing"

func TestScaledAmount(t *testing.T) {
	cases := []struct {
		name         string
		amount       float64
		servings     int
		household    int
		pairRowCount int
		want         float64
		wantScaled   bool
	}{
		{"household が 0 なら量をそのまま返し、換算していないことを返す", 3, 2, 0, 1, 3, false},
		{"pairRowCount が 1 なら amount × household / servings を返す", 3, 2, 4, 1, 6, true},
		{"household が servings より少なければ減らす", 3, 4, 1, 1, 0.75, true},
		{"pairRowCount が 2 なら量をそのまま返す", 3, 2, 4, 2, 3, false},
		{"amount が 0 なら 0 を返し、換算していないことを返す", 0, 2, 4, 1, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, scaled := ScaledAmount(tc.amount, tc.servings, tc.household, tc.pairRowCount)
			if got != tc.want || scaled != tc.wantScaled {
				t.Errorf("ScaledAmount(%v, %d, %d, %d) = (%v, %v), want (%v, %v)",
					tc.amount, tc.servings, tc.household, tc.pairRowCount, got, scaled, tc.want, tc.wantScaled)
			}
		})
	}
}
