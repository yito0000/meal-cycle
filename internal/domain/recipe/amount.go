package recipe

import (
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/text/unicode/norm"
	"golang.org/x/text/width"
)

var (
	amountMixedRe    = regexp.MustCompile(`^(\d+)(?:と|\s+)(\d+)/(\d+)`)
	amountRangeRe    = regexp.MustCompile(`^(\d+(?:\.\d+)?(?:/\d+)?)\s*[~\-]\s*(\d+(?:\.\d+)?(?:/\d+)?)`)
	amountFractionRe = regexp.MustCompile(`^(\d+)/(\d+)`)
	amountDecimalRe  = regexp.MustCompile(`^\d+(?:\.\d+)?`)
)

// ParseAmount はインポートするファイルの量の表記ゆれを、量と単位に直す (add-recipe-bulk-import design.md D5)。
// raw は YAML が読んだ値をそのまま受ける。読めないときは ok=false を返す。
func ParseAmount(raw any, unit string, measuringUnits, unquantifiedWords []string) (float64, string, bool) {
	unit = strings.TrimSpace(unit)
	switch v := raw.(type) {
	case nil:
		return 0, unit, true
	case int:
		return roundAmount(float64(v), unit)
	case int64:
		return roundAmount(float64(v), unit)
	case uint64:
		return roundAmount(float64(v), unit)
	case float64:
		return roundAmount(v, unit)
	case string:
		return parseAmountString(v, unit, measuringUnits, unquantifiedWords)
	default:
		return 0, "", false
	}
}

func parseAmountString(raw, unit string, measuringUnits, unquantifiedWords []string) (float64, string, bool) {
	// Narrow は全角カタカナまで半角カナにして「カップ」を壊すので Fold を使う。
	// Fold は半角カナの濁点を結合文字で残すので、NFC で「プ」1 文字に合成する。
	s := norm.NFC.String(width.Fold.String(raw))
	// 波ダッシュ (U+301C) は width.Fold の対象外なので個別に揃える。
	s = strings.NewReplacer("〜", "~", "～", "~").Replace(s)
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, unit, true
	}
	if slices.Contains(unquantifiedWords, s) {
		if unit == "" {
			return 0, s, true
		}
		return 0, unit, true
	}

	prefix := ""
	for _, u := range measuringUnits {
		if u != "" && strings.HasPrefix(s, u) {
			prefix = u
			s = strings.TrimSpace(strings.TrimPrefix(s, u))
			break
		}
	}

	// 範囲は分数より先に読む。「1/2~1」を分数 + 単位「~1」と読まないため。
	var (
		amount float64
		rest   string
		ok     bool
	)
	if m := amountMixedRe.FindStringSubmatch(s); m != nil {
		amount, ok = mixedValue(m[1], m[2], m[3])
		rest = s[len(m[0]):]
	} else if m := amountRangeRe.FindStringSubmatch(s); m != nil {
		lo, okLo := simpleValue(m[1])
		hi, okHi := simpleValue(m[2])
		// S07 で買い足りなくなるのを避けるため、範囲は大きいほうを取る。
		amount, ok = math.Max(lo, hi), okLo && okHi
		rest = s[len(m[0]):]
	} else if m := amountFractionRe.FindStringSubmatch(s); m != nil {
		amount, ok = simpleValue(m[0])
		rest = s[len(m[0]):]
	} else if m := amountDecimalRe.FindString(s); m != "" {
		amount, ok = simpleValue(m)
		rest = s[len(m):]
	}
	if !ok {
		return 0, "", false
	}

	suffix := strings.TrimSpace(rest)
	if prefix != "" && suffix != "" {
		return 0, "", false
	}
	candidate := prefix + suffix
	if candidate != "" {
		if unit == "" {
			unit = candidate
		} else if unit != candidate {
			return 0, "", false
		}
	}
	return roundAmount(amount, unit)
}

func mixedValue(whole, num, den string) (float64, bool) {
	w, err := strconv.ParseFloat(whole, 64)
	if err != nil {
		return 0, false
	}
	f, ok := simpleValue(num + "/" + den)
	return w + f, ok
}

// 小数か分数 1 つを読む。
func simpleValue(s string) (float64, bool) {
	if num, den, found := strings.Cut(s, "/"); found {
		n, errN := strconv.ParseFloat(num, 64)
		d, errD := strconv.ParseFloat(den, 64)
		if errN != nil || errD != nil || d == 0 {
			return 0, false
		}
		return n / d, true
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

func roundAmount(v float64, unit string) (float64, string, bool) {
	if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, "", false
	}
	return math.Round(v*100) / 100, unit, true
}
