package mealplan

import (
	"strings"
	"testing"
)

func TestNewBatchID(t *testing.T) {
	cases := []struct {
		name  string
		check func(t *testing.T, got string)
	}{
		{
			name: "returns a non-empty string",
			check: func(t *testing.T, got string) {
				if got == "" {
					t.Errorf("NewBatchID returned empty string")
				}
			},
		},
		{
			name: "returned value length is at least 32 hex chars (UUID v7 相当)",
			check: func(t *testing.T, got string) {
				// UUID v7 の文字列表現 (ハイフン込み) は 36 文字、ハイフンを除いても 32 文字。
				stripped := strings.ReplaceAll(got, "-", "")
				if len(stripped) < 32 {
					t.Errorf("stripped length = %d, want >= 32 (got %q)", len(stripped), got)
				}
			},
		},
		{
			name: "returned value uses only hex characters (after removing hyphens)",
			check: func(t *testing.T, got string) {
				stripped := strings.ReplaceAll(got, "-", "")
				for _, c := range stripped {
					isDigit := c >= '0' && c <= '9'
					isHexAlpha := c >= 'a' && c <= 'f'
					if !isDigit && !isHexAlpha {
						t.Errorf("non-hex char %q in %q", c, got)
						return
					}
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NewBatchID()
			tc.check(t, got)
		})
	}

	t.Run("returns distinct values across calls", func(t *testing.T) {
		seen := map[string]bool{}
		for i := 0; i < 8; i++ {
			id := NewBatchID()
			if seen[id] {
				t.Fatalf("NewBatchID returned duplicate %q on call %d", id, i)
			}
			seen[id] = true
		}
	})
}
