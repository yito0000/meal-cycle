package recipe

import (
	"reflect"
	"strings"
	"testing"
)

func readBack(t *testing.T, content string) Recipe {
	t.Helper()
	got, issue, arg := ParseImportFile(content, 2, []string{"大さじ", "小さじ", "カップ"}, []string{"少々", "適量"})
	if issue != ImportIssueNone {
		t.Fatalf("ParseImportFile issue %q (%q)\n%s", issue, arg, content)
	}
	return got
}

func TestFormatExportFile(t *testing.T) {
	cases := []struct {
		name  string
		input Recipe
		check func(t *testing.T, content string, got Recipe)
	}{
		{
			name: "all fields read back as the same values",
			input: Recipe{
				Name: "肉じゃが", Servings: 3, Tags: []string{"和食", "煮物"},
				Ingredients: []RecipeIngredient{{Name: "じゃがいも", Amount: 3, Unit: "個"}, {Name: "醤油", Amount: 2, Unit: "大さじ"}},
				Steps:       "1. 切る\n2. 煮る", Notes: "弱火で",
			},
			check: func(t *testing.T, _ string, got Recipe) {
				if got.Name != "肉じゃが" || got.Servings != 3 {
					t.Errorf("name/servings: got %q/%d", got.Name, got.Servings)
				}
				if !reflect.DeepEqual(got.Tags, []string{"和食", "煮物"}) {
					t.Errorf("tags: got %v", got.Tags)
				}
				want := []RecipeIngredient{{Name: "じゃがいも", Amount: 3, Unit: "個"}, {Name: "醤油", Amount: 2, Unit: "大さじ"}}
				if !reflect.DeepEqual(got.Ingredients, want) {
					t.Errorf("ingredients: got %v, want %v", got.Ingredients, want)
				}
				if got.Steps != "1. 切る\n2. 煮る" || got.Notes != "弱火で" {
					t.Errorf("steps/notes: got %q/%q", got.Steps, got.Notes)
				}
			},
		},
		{
			name:  "empty tags and ingredients are omitted from the frontmatter",
			input: Recipe{Name: "x", Servings: 1, Tags: []string{}, Ingredients: []RecipeIngredient{}},
			check: func(t *testing.T, content string, _ Recipe) {
				if strings.Contains(content, "tags") || strings.Contains(content, "ingredients") {
					t.Errorf("want no tags/ingredients keys:\n%s", content)
				}
			},
		},
		{
			name:  "only steps writes only the steps section",
			input: Recipe{Name: "x", Servings: 1, Steps: "煮る"},
			check: func(t *testing.T, content string, got Recipe) {
				if !strings.Contains(content, "## 手順") || strings.Contains(content, "## メモ") {
					t.Errorf("sections:\n%s", content)
				}
				if got.Steps != "煮る" || got.Notes != "" {
					t.Errorf("steps/notes: got %q/%q", got.Steps, got.Notes)
				}
			},
		},
		{
			name:  "only notes writes only the notes section",
			input: Recipe{Name: "x", Servings: 1, Notes: "弱火で"},
			check: func(t *testing.T, content string, got Recipe) {
				if strings.Contains(content, "## 手順") || !strings.Contains(content, "## メモ") {
					t.Errorf("sections:\n%s", content)
				}
				if got.Steps != "" || got.Notes != "弱火で" {
					t.Errorf("steps/notes: got %q/%q", got.Steps, got.Notes)
				}
			},
		},
		{
			name:  "no steps and notes writes no body",
			input: Recipe{Name: "x", Servings: 1},
			check: func(t *testing.T, content string, got Recipe) {
				if strings.Contains(content, "##") || !strings.HasSuffix(content, "---\n") {
					t.Errorf("want no body:\n%s", content)
				}
				if got.Steps != "" || got.Notes != "" {
					t.Errorf("steps/notes: got %q/%q", got.Steps, got.Notes)
				}
			},
		},
		{
			name:  "zero amount with unit 少々 reads back as zero and 少々",
			input: Recipe{Name: "x", Servings: 1, Ingredients: []RecipeIngredient{{Name: "塩", Amount: 0, Unit: "少々"}}},
			check: func(t *testing.T, _ string, got Recipe) {
				want := []RecipeIngredient{{Name: "塩", Amount: 0, Unit: "少々"}}
				if !reflect.DeepEqual(got.Ingredients, want) {
					t.Errorf("ingredients: got %v, want %v", got.Ingredients, want)
				}
			},
		},
		{
			name: "fractional amounts read back unchanged",
			input: Recipe{Name: "x", Servings: 1, Ingredients: []RecipeIngredient{
				{Name: "玉ねぎ", Amount: 0.5, Unit: "個"}, {Name: "牛乳", Amount: 0.33, Unit: "カップ"},
			}},
			check: func(t *testing.T, _ string, got Recipe) {
				want := []RecipeIngredient{{Name: "玉ねぎ", Amount: 0.5, Unit: "個"}, {Name: "牛乳", Amount: 0.33, Unit: "カップ"}}
				if !reflect.DeepEqual(got.Ingredients, want) {
					t.Errorf("ingredients: got %v, want %v", got.Ingredients, want)
				}
			},
		},
		{
			name:  "names and tags with colon, hash and leading dash read back unchanged",
			input: Recipe{Name: "- 鶏: 照り焼き #1", Servings: 1, Tags: []string{"#和食", "- 主菜: 肉"}},
			check: func(t *testing.T, _ string, got Recipe) {
				if got.Name != "- 鶏: 照り焼き #1" {
					t.Errorf("name: got %q", got.Name)
				}
				if !reflect.DeepEqual(got.Tags, []string{"#和食", "- 主菜: 肉"}) {
					t.Errorf("tags: got %v", got.Tags)
				}
			},
		},
		{
			name:  "level 2 heading in steps is demoted and the following steps survive",
			input: Recipe{Name: "x", Servings: 1, Steps: "## 下ごしらえ\n切る\n## 仕上げ\n煮る"},
			check: func(t *testing.T, _ string, got Recipe) {
				if got.Steps != "### 下ごしらえ\n切る\n### 仕上げ\n煮る" {
					t.Errorf("steps: got %q", got.Steps)
				}
			},
		},
		{
			name:  "lines inside a code block are not demoted",
			input: Recipe{Name: "x", Servings: 1, Steps: "```\n## コード\n```\n~~~\n## チルダ\n~~~"},
			check: func(t *testing.T, _ string, got Recipe) {
				if got.Steps != "```\n## コード\n```\n~~~\n## チルダ\n~~~" {
					t.Errorf("steps: got %q", got.Steps)
				}
			},
		},
		{
			name:  "level 2 heading in notes is demoted too",
			input: Recipe{Name: "x", Servings: 1, Notes: "## 補足\n冷凍できる"},
			check: func(t *testing.T, _ string, got Recipe) {
				if got.Notes != "### 補足\n冷凍できる" {
					t.Errorf("notes: got %q", got.Notes)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := FormatExportFile(tc.input)
			tc.check(t, content, readBack(t, content))
		})
	}
}

func TestExportFileName(t *testing.T) {
	long := strings.Repeat("あ", 120)
	cases := []struct {
		name  string
		input []string
		want  []string
	}{
		{name: "plain name gets .md", input: []string{"肉じゃが"}, want: []string{"肉じゃが.md"}},
		{name: "slash becomes underscore", input: []string{"鶏/豚の生姜焼き"}, want: []string{"鶏_豚の生姜焼き.md"}},
		{name: "every forbidden and control character becomes underscore", input: []string{"a<b>c:d\"e\\f|g?h*i\tj\x01k"}, want: []string{"a_b_c_d_e_f_g_h_i_j_k.md"}},
		{name: "surrounding spaces and trailing dots are removed", input: []string{"  カレー. . "}, want: []string{"カレー.md"}},
		{name: "names over 100 runes are truncated", input: []string{long}, want: []string{strings.Repeat("あ", 100) + ".md"}},
		{name: "empty result falls back to recipe", input: []string{" ... "}, want: []string{"recipe.md"}},
		{name: "reserved device names get a leading underscore", input: []string{"CON", "com1"}, want: []string{"_CON.md", "_com1.md"}},
		{name: "same name is numbered from 2", input: []string{"カレー", "カレー", "カレー"}, want: []string{"カレー.md", "カレー-2.md", "カレー-3.md"}},
		{name: "names differing only in case are numbered", input: []string{"Curry", "curry"}, want: []string{"Curry.md", "curry-2.md"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			used := map[string]bool{}
			for i, in := range tc.input {
				if got := ExportFileName(in, used); got != tc.want[i] {
					t.Errorf("ExportFileName(%q) = %q, want %q", in, got, tc.want[i])
				}
			}
		})
	}
}
