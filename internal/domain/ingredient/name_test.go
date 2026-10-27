package ingredient

import "testing"

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"前後の空白を除く", "  玉ねぎ ", "玉ねぎ"},
		{"全角英字を半角にする", "ｇ", "g"},
		{"全角数字と全角空白を半角にする", "１２３　４", "123 4"},
		{"半角カタカナをひらがなにして濁点を合成する", "ﾀﾏﾈｷﾞ", "たまねぎ"},
		{"カタカナをひらがなにする", "タマネギ", "たまねぎ"},
		{"長音符を残す", "ピーマン", "ぴーまん"},
		{"英字の大文字を小文字にする", "KG", "kg"},
		{"漢字は変えない", "玉ねぎ", "玉ねぎ"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeName(tc.in); got != tc.want {
				t.Errorf("NormalizeName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
