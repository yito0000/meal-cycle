// Package ingredient は食材名を扱う capability 横断のドメイン関数を置く。
package ingredient

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// NormalizeName は食材名を比較用にそろえる (ひらがなとカタカナ、全角と半角、英字の大小)。
// 漢字とかなの違い (玉ねぎ / たまねぎ) は辞書が無いとそろえられないので扱わない (review R19 の別名の辞書に回す)。
func NormalizeName(s string) string {
	s = norm.NFKC.String(strings.TrimSpace(s))
	s = strings.Map(func(r rune) rune {
		if r >= 'ァ' && r <= 'ヶ' {
			return r - 0x60
		}
		return r
	}, s)
	return strings.ToLower(s)
}
