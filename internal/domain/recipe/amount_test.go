package recipe

import "testing"

func TestParseAmount(t *testing.T) {
	measuring := []string{"大さじ", "小さじ", "カップ"}
	unquantified := []string{"少々", "適量", "適宜", "少量", "ひとつまみ", "お好みで"}

	cases := []struct {
		name       string
		raw        any
		unit       string
		wantAmount float64
		wantUnit   string
		wantOK     bool
	}{
		{"int keeps the unit", 2, "個", 2, "個", true},
		{"float is the amount as is", 0.5, "", 0.5, "", true},
		{"negative number is rejected", -1, "", 0, "", false},
		{"nil is zero", nil, "個", 0, "個", true},
		{"empty string is zero", "", "", 0, "", true},
		{"fraction becomes decimal", "1/2", "個", 0.5, "個", true},
		{"fraction is rounded to 2 decimals", "1/3", "", 0.33, "", true},
		{"mixed fraction with と", "1と1/2", "カップ", 1.5, "カップ", true},
		{"mixed fraction with space", "1 1/2", "カップ", 1.5, "カップ", true},
		{"full-width decimal", "１．５", "", 1.5, "", true},
		{"full-width fraction", "１／２", "", 0.5, "", true},
		{"range with wave dash takes the larger", "2〜3", "", 3, "", true},
		{"range with tilde takes the larger", "2~3", "", 3, "", true},
		{"range with hyphen takes the larger", "2-3", "", 3, "", true},
		{"measuring unit prefix", "大さじ2", "", 2, "大さじ", true},
		{"unit suffix", "3個", "", 3, "個", true},
		{"katakana unit suffix stays full-width", "2カップ", "", 2, "カップ", true},
		{"katakana measuring unit prefix", "カップ1/2", "", 0.5, "カップ", true},
		{"half-width katakana unit becomes full-width", "2ｶｯﾌﾟ", "", 2, "カップ", true},
		{"prefix with mixed fraction", "大さじ1と1/2", "", 1.5, "大さじ", true},
		{"unit suffix matching unit", "3個", "個", 3, "個", true},
		{"unit suffix conflicting unit", "3個", "本", 0, "", false},
		{"unit on both sides is rejected", "大さじ2杯", "", 0, "", false},
		{"unquantified word becomes the unit", "少々", "", 0, "少々", true},
		{"unquantified word yields to unit", "適量", "g", 0, "g", true},
		{"unreadable word is rejected", "たくさん", "", 0, "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotAmount, gotUnit, gotOK := ParseAmount(tc.raw, tc.unit, measuring, unquantified)
			if gotOK != tc.wantOK {
				t.Fatalf("ok: got %v, want %v", gotOK, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if gotAmount != tc.wantAmount {
				t.Errorf("amount: got %v, want %v", gotAmount, tc.wantAmount)
			}
			if gotUnit != tc.wantUnit {
				t.Errorf("unit: got %q, want %q", gotUnit, tc.wantUnit)
			}
		})
	}
}
