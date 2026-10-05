package shopping

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// 漢字とかなの違い (玉ねぎ / たまねぎ) は辞書が無いとそろえられないので扱わない (review R19 の別名の辞書に回す)。
func normalizeKey(s string) string {
	s = norm.NFKC.String(strings.TrimSpace(s))
	s = strings.Map(func(r rune) rune {
		if r >= 'ァ' && r <= 'ヶ' {
			return r - 0x60
		}
		return r
	}, s)
	return strings.ToLower(s)
}
