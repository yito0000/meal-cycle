package recipe

import (
	"bytes"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// ExportResult は書き出しの結果。ダイアログを閉じたときは Cancelled だけが true になる。
type ExportResult struct {
	Cancelled bool `json:"cancelled"`
	// 作った書き出し用のフォルダの絶対パス。
	Dir   string `json:"dir"`
	Count int    `json:"count"`
}

// ファイル名の上限。Windows のパス長 (260) に、選んだフォルダのパスの分の余裕を残す (add-recipe-export D4)。
const exportFileNameMaxRunes = 100

type exportFrontmatter struct {
	Name        string             `yaml:"name"`
	Servings    int                `yaml:"servings"`
	Tags        []string           `yaml:"tags,omitempty"`
	Ingredients []exportIngredient `yaml:"ingredients,omitempty"`
}

// Amount に omitempty を付けないのは、量 0 を書いて読み戻しで文字列の量と解釈される余地を作らないため (add-recipe-export D3)。
type exportIngredient struct {
	Name   string  `yaml:"name"`
	Amount float64 `yaml:"amount"`
	Unit   string  `yaml:"unit,omitempty"`
}

// FormatExportFile は 1 件をインポートと同じ形式の Markdown にする (add-recipe-export D3)。
func FormatExportFile(r Recipe) string {
	fm := exportFrontmatter{
		Name:        r.Name,
		Servings:    r.Servings,
		Tags:        r.Tags,
		Ingredients: make([]exportIngredient, 0, len(r.Ingredients)),
	}
	for _, ing := range r.Ingredients {
		fm.Ingredients = append(fm.Ingredients, exportIngredient(ing))
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	// 書き込み先がメモリなので、struct が固定の形である限り失敗しない。
	_ = enc.Encode(fm)
	_ = enc.Close()

	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(buf.String())
	b.WriteString("---\n")
	// 見出しの語は表示文言ではなく、インポート形式のキーワード。
	writeExportSection(&b, "手順", r.Steps)
	writeExportSection(&b, "メモ", r.Notes)
	return b.String()
}

func writeExportSection(b *strings.Builder, title, body string) {
	if strings.TrimSpace(body) == "" {
		return
	}
	b.WriteString("\n## " + title + "\n\n")
	b.WriteString(demoteLevel2Headings(body))
	b.WriteString("\n")
}

// インポートはレベル 2 の見出しを節の区切りに読むので、本文中のものは 1 段下げて内容が欠けないようにする。
func demoteLevel2Headings(body string) string {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	inFence := false
	for i, line := range lines {
		if isFenceLine(line) {
			inFence = !inFence
		}
		if !inFence && strings.HasPrefix(line, "## ") {
			lines[i] = "#" + line
		}
	}
	return strings.Join(lines, "\n")
}

// ExportFileName はレシピ名からファイル名を作る。どの OS でも同じ名前になるよう Windows の規則に合わせる (add-recipe-export D4)。
// used には小文字にした拡張子なしの名前が溜まる。Windows と macOS は大文字と小文字を区別しないため。
func ExportFileName(name string, used map[string]bool) string {
	base := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, name)
	base = trimExportFileName(base)
	if runes := []rune(base); len(runes) > exportFileNameMaxRunes {
		base = trimExportFileName(string(runes[:exportFileNameMaxRunes]))
	}
	if base == "" {
		base = "recipe"
	}
	if isWindowsReservedName(base) {
		base = "_" + base
	}

	candidate := base
	for n := 2; used[strings.ToLower(candidate)]; n++ {
		candidate = base + "-" + strconv.Itoa(n)
	}
	used[strings.ToLower(candidate)] = true
	return candidate + ".md"
}

// Windows は末尾の空白と . を黙って落とすので、残すと書いた名前と実際の名前がずれる。
func trimExportFileName(s string) string {
	for {
		t := strings.TrimRight(strings.TrimSpace(s), ".")
		if t == s {
			return s
		}
		s = t
	}
}

// Windows は「CON.md」のように拡張子が付いても予約名として扱う。
func isWindowsReservedName(base string) bool {
	stem, _, _ := strings.Cut(base, ".")
	switch stem = strings.ToUpper(strings.TrimSpace(stem)); stem {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(stem) == 4 && (strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT")) {
		return stem[3] >= '1' && stem[3] <= '9'
	}
	return false
}
