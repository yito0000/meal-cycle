package recipe

import (
	"reflect"
	"testing"
)

func TestParseImportFile(t *testing.T) {
	measuring := []string{"大さじ", "小さじ", "カップ"}
	unquantified := []string{"少々", "適量"}
	const defaultServings = 2

	cases := []struct {
		name         string
		content      string
		wantIssue    ImportIssue
		wantIssueArg string
		check        func(t *testing.T, got Recipe)
	}{
		{
			name: "frontmatter fields become recipe fields",
			content: `---
name: "肉じゃが"
servings: 3
tags: ["和食", "煮物"]
ingredients:
  - name: "じゃがいも"
    amount: 3
    unit: "個"
  - name: "醤油"
    amount: "大さじ2"
---
## 手順
煮る
`,
			check: func(t *testing.T, got Recipe) {
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
			},
		},
		{
			name:    "missing servings falls back to the default",
			content: "---\nname: カレー\n---\n",
			check: func(t *testing.T, got Recipe) {
				if got.Servings != defaultServings {
					t.Errorf("servings: got %d, want %d", got.Servings, defaultServings)
				}
			},
		},
		{
			name:    "steps and notes sections exclude their heading lines",
			content: "---\nname: x\n---\n## 手順\n\n1. 切る\n2. 煮る\n\n## メモ\n弱火で\n",
			check: func(t *testing.T, got Recipe) {
				wantSteps(t, got, "1. 切る\n2. 煮る", "弱火で")
			},
		},
		{
			name:    "Steps and notes headings in English are case-insensitive",
			content: "---\nname: x\n---\n## Steps\ncut\n## notes\nlow heat\n",
			check: func(t *testing.T, got Recipe) {
				wantSteps(t, got, "cut", "low heat")
			},
		},
		{
			name:    "level 3 headings stay inside the steps",
			content: "---\nname: x\n---\n## 手順\n煮る\n### コツ\n弱火\n",
			check: func(t *testing.T, got Recipe) {
				wantSteps(t, got, "煮る\n### コツ\n弱火", "")
			},
		},
		{
			name:    "body without headings becomes the steps",
			content: "---\nname: x\n---\n\n切って煮る\n\n",
			check: func(t *testing.T, got Recipe) {
				wantSteps(t, got, "切って煮る", "")
			},
		},
		{
			name:    "text before the first heading is dropped when a heading exists",
			content: "---\nname: x\n---\n前置き\n## 手順\n煮る\n",
			check: func(t *testing.T, got Recipe) {
				wantSteps(t, got, "煮る", "")
			},
		},
		{
			name:    "URL in the notes is kept as is",
			content: "---\nname: x\n---\n## メモ\n出典: https://example.com/recipes/1?a=b#c\n",
			check: func(t *testing.T, got Recipe) {
				wantSteps(t, got, "", "出典: https://example.com/recipes/1?a=b#c")
			},
		},
		{
			name:      "content not starting with --- has no frontmatter",
			content:   "# 肉じゃが\nname: x\n",
			wantIssue: ImportIssueNoFrontmatter,
		},
		{
			name:      "missing closing --- has no frontmatter",
			content:   "---\nname: x\n## 手順\n",
			wantIssue: ImportIssueNoFrontmatter,
		},
		{
			name:      "broken YAML is invalid_yaml",
			content:   "---\nname: [x\n---\n",
			wantIssue: ImportIssueInvalidYAML,
		},
		{
			name:         "unreadable amount returns amount_invalid with the ingredient name",
			content:      "---\nname: x\ningredients:\n  - name: にんじん\n    amount: たくさん\n---\n",
			wantIssue:    ImportIssueAmountInvalid,
			wantIssueArg: "にんじん",
		},
		{
			name:    "unknown frontmatter fields are ignored",
			content: "---\nname: x\nsource: https://example.com\ncookTime: 30\n---\n",
			check: func(t *testing.T, got Recipe) {
				if got.Name != "x" {
					t.Errorf("name: got %q", got.Name)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, issue, arg := ParseImportFile(tc.content, defaultServings, measuring, unquantified)
			if issue != tc.wantIssue {
				t.Fatalf("issue: got %q, want %q", issue, tc.wantIssue)
			}
			if arg != tc.wantIssueArg {
				t.Errorf("issueArg: got %q, want %q", arg, tc.wantIssueArg)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

func wantSteps(t *testing.T, got Recipe, steps, notes string) {
	t.Helper()
	if got.Steps != steps {
		t.Errorf("steps: got %q, want %q", got.Steps, steps)
	}
	if got.Notes != notes {
		t.Errorf("notes: got %q, want %q", got.Notes, notes)
	}
}
