package recipe

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// ImportFile はフロントが読んだ 1 ファイル。Go 側はファイルシステムに触れない (add-recipe-bulk-import design.md D2)。
type ImportFile struct {
	// フォルダ選択なら相対パス。表示と並び順に使う。
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ImportIssue はファイルごとの問題のコード。Go 側は表示文言を持たず、フロントが i18n key に対応付ける (design.md D4)。
type ImportIssue string

const (
	ImportIssueNone                ImportIssue = ""
	ImportIssueTooLarge            ImportIssue = "too_large"
	ImportIssueNoFrontmatter       ImportIssue = "no_frontmatter"
	ImportIssueInvalidYAML         ImportIssue = "invalid_yaml"
	ImportIssueNameEmpty           ImportIssue = "name_empty"
	ImportIssueNameTooLong         ImportIssue = "name_too_long"
	ImportIssueServingsOutOfRange  ImportIssue = "servings_out_of_range"
	ImportIssueStepsTooLong        ImportIssue = "steps_too_long"
	ImportIssueNotesTooLong        ImportIssue = "notes_too_long"
	ImportIssueTagTooLong          ImportIssue = "tag_too_long"
	ImportIssueIngredientNameEmpty ImportIssue = "ingredient_name_empty"
	ImportIssueAmountInvalid       ImportIssue = "amount_invalid"
)

// ImportCandidate は取り込む前の確認に出す 1 ファイル分の結果。
type ImportCandidate struct {
	FileName string      `json:"fileName"`
	Recipe   Recipe      `json:"recipe"`
	Issue    ImportIssue `json:"issue"`
	// 問題の材料名など、メッセージに差し込む値。
	IssueArg  string `json:"issueArg"`
	Duplicate bool   `json:"duplicate"`
}

type importFrontmatter struct {
	Name        string   `yaml:"name"`
	Servings    *int     `yaml:"servings"`
	Tags        []string `yaml:"tags"`
	Ingredients []struct {
		Name   string `yaml:"name"`
		Amount any    `yaml:"amount"`
		Unit   string `yaml:"unit"`
	} `yaml:"ingredients"`
}

// ParseImportFile は 1 つの Markdown ファイルをレシピに読む (design.md D1)。
// 入力制約の検証はサービス層の validateRecipeInput に任せ、ここでは形式と量の読み取りだけを見る。
func ParseImportFile(content string, defaultServings int, measuringUnits, unquantifiedWords []string) (Recipe, ImportIssue, string) {
	content = strings.TrimPrefix(content, "\ufeff")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return Recipe{}, ImportIssueNoFrontmatter, ""
	}
	closing := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t") == "---" {
			closing = i
			break
		}
	}
	if closing < 0 {
		return Recipe{}, ImportIssueNoFrontmatter, ""
	}

	var fm importFrontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:closing], "\n")), &fm); err != nil {
		return Recipe{}, ImportIssueInvalidYAML, ""
	}

	rec := Recipe{
		Name:        fm.Name,
		Servings:    defaultServings,
		Tags:        fm.Tags,
		Ingredients: make([]RecipeIngredient, 0, len(fm.Ingredients)),
	}
	if fm.Servings != nil {
		rec.Servings = *fm.Servings
	}
	for _, ing := range fm.Ingredients {
		// 名前の無い材料は量を読まずに残し、サービス層の検証で何番目かを添えて落とす。
		if strings.TrimSpace(ing.Name) == "" {
			rec.Ingredients = append(rec.Ingredients, RecipeIngredient{Name: ing.Name, Unit: ing.Unit})
			continue
		}
		amount, unit, ok := ParseAmount(ing.Amount, ing.Unit, measuringUnits, unquantifiedWords)
		if !ok {
			return Recipe{}, ImportIssueAmountInvalid, strings.TrimSpace(ing.Name)
		}
		rec.Ingredients = append(rec.Ingredients, RecipeIngredient{Name: ing.Name, Amount: amount, Unit: unit})
	}

	rec.Steps, rec.Notes = splitImportBody(lines[closing+1:])
	return rec, ImportIssueNone, ""
}

type importSection int

const (
	importSectionNone importSection = iota
	importSectionSteps
	importSectionNotes
	importSectionOther
)

// レベル 2 の見出しだけを区切りにする。コードブロック内の「## 」は見出しとみなさない。
func splitImportBody(lines []string) (string, string) {
	var steps, notes []string
	section := importSectionNone
	hasHeading := false
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
		}
		if !inFence && strings.HasPrefix(line, "## ") {
			switch title := strings.TrimSpace(line[3:]); {
			case title == "手順" || strings.EqualFold(title, "steps"):
				section, hasHeading = importSectionSteps, true
				continue
			case title == "メモ" || strings.EqualFold(title, "notes"):
				section, hasHeading = importSectionNotes, true
				continue
			default:
				section = importSectionOther
				continue
			}
		}
		switch section {
		case importSectionSteps:
			steps = append(steps, line)
		case importSectionNotes:
			notes = append(notes, line)
		}
	}
	if !hasHeading {
		return trimBlankLines(lines), ""
	}
	return trimBlankLines(steps), trimBlankLines(notes)
}

func trimBlankLines(lines []string) string {
	start, end := 0, len(lines)
	for start < end && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return strings.Join(lines[start:end], "\n")
}
