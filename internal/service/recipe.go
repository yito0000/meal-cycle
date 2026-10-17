package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yito0000/meal-cycle/internal/domain/recipe"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// バリデーション上限。ユーザーが変えたい値ではなく実装上限なので、可変値集約の
// defaults.go には置かない (design.md D12)。
const (
	recipeNameMaxRunes  = 200
	recipeServingsMin   = 1
	recipeServingsMax   = 99
	recipeStepsMaxRunes = 4000
	recipeNotesMaxRunes = 2000
	recipeTagMaxRunes   = 30
	// 冷蔵庫の在庫の上限 (stockAmountMax) と揃える。S07 の一括登録で必要量をそのまま在庫に入れるので、変えるときは一緒に変える。
	recipeAmountMax = 99999
)

// インポートの上限。異常なフォルダを選んだときの歯止めで、ユーザーが変える値ではない (add-recipe-bulk-import D2)。
const (
	recipeImportMaxFiles = 100
	recipeImportMaxBytes = 64 * 1024
)

// 入力バリデーションエラー。フロントで i18n key に対応付ける (design.md D12)。
var (
	ErrRecipeNameTooLong                = errors.New("recipe name too long")
	ErrRecipeServingsOutOfRange         = errors.New("recipe servings out of range")
	ErrRecipeIngredientNameEmpty        = errors.New("recipe ingredient name empty")
	ErrRecipeIngredientAmountOutOfRange = errors.New("recipe ingredient amount out of range")
	ErrRecipeStepsTooLong               = errors.New("recipe steps too long")
	ErrRecipeNotesTooLong               = errors.New("recipe notes too long")
	ErrRecipeTagTooLong                 = errors.New("recipe tag too long")
	ErrRecipeImportTooManyFiles         = errors.New("recipe import too many files")
	// ダイアログを開けなかったときもこれに寄せる。利用者から見てどちらも「書き出せなかった」で足りるため (add-recipe-export D6)。
	ErrRecipeExportWrite = errors.New("recipe export write failed")
	ErrRecipeExportEmpty = errors.New("recipe export empty")
)

// validateRecipeInput のエラーをインポートの問題コードに引く (add-recipe-bulk-import D4)。
var recipeImportIssueByErr = []struct {
	err   error
	issue recipe.ImportIssue
}{
	{sqlite.ErrRecipeNameEmpty, recipe.ImportIssueNameEmpty},
	{ErrRecipeNameTooLong, recipe.ImportIssueNameTooLong},
	{ErrRecipeServingsOutOfRange, recipe.ImportIssueServingsOutOfRange},
	{ErrRecipeStepsTooLong, recipe.ImportIssueStepsTooLong},
	{ErrRecipeNotesTooLong, recipe.ImportIssueNotesTooLong},
	{ErrRecipeTagTooLong, recipe.ImportIssueTagTooLong},
	{ErrRecipeIngredientNameEmpty, recipe.ImportIssueIngredientNameEmpty},
	{ErrRecipeIngredientAmountOutOfRange, recipe.ImportIssueAmountInvalid},
}

// RecipeService はレシピの List / GetByID / Create / Update / Delete をフロントに公開する。
// Create / Update は保存前に validateRecipeInput を走らせ、違反時は Repository
// に到達せず型付きエラーを返す。
type RecipeService struct {
	repo   *sqlite.RecipeRepo
	cfg    configSource
	picker FolderPicker
	now    func() time.Time
}

// picker は書き出しでしか使わないので、書き出しを呼ばない呼び出し元は nil を渡してよい。
func NewRecipeService(db *sql.DB, cfg configSource, picker FolderPicker) *RecipeService {
	return &RecipeService{
		repo:   sqlite.NewRecipeRepo(db),
		cfg:    cfg,
		picker: picker,
		now:    time.Now,
	}
}

// List は S02 一覧向けの projection。Ingredients / Steps / Notes は空値で、
// IngredientCount のみ集計して返す (詳細は sqlite.RecipeRepo.List)。
func (s *RecipeService) List(ctx context.Context) ([]recipe.Recipe, error) {
	return s.repo.List(ctx)
}

func (s *RecipeService) GetByID(ctx context.Context, id string) (recipe.Recipe, error) {
	return s.repo.GetByID(ctx, id)
}

// Create は input からレシピを新規作成する。input.ID / CreatedAt / UpdatedAt /
// IngredientCount は Repository 側で決めるため入力側の値は無視される。
func (s *RecipeService) Create(ctx context.Context, input recipe.Recipe) (recipe.Recipe, error) {
	normalized, err := validateRecipeInput(input)
	if err != nil {
		return recipe.Recipe{}, err
	}
	return s.repo.Create(ctx, normalized)
}

// Update は指定 ID のレシピを上書きする。updated_at は現在時刻に進む。
// input.ID / CreatedAt / UpdatedAt / IngredientCount は Create 同様に無視。
func (s *RecipeService) Update(ctx context.Context, id string, input recipe.Recipe) (recipe.Recipe, error) {
	normalized, err := validateRecipeInput(input)
	if err != nil {
		return recipe.Recipe{}, err
	}
	return s.repo.Update(ctx, id, normalized)
}

// Delete は論理削除する。献立の履歴を残すため、そのレシピを使った献立は消えない (add-recipe-deletion D1)。
// 削除済みのレシピに対しては何もせず nil を返す。
func (s *RecipeService) Delete(ctx context.Context, id string) error {
	return s.repo.Archive(ctx, id)
}

// PreviewImport はファイルを読んで取り込む前の確認に出す候補を返す。DB には何も書かない (add-recipe-bulk-import D3)。
func (s *RecipeService) PreviewImport(ctx context.Context, files []recipe.ImportFile) ([]recipe.ImportCandidate, error) {
	if len(files) > recipeImportMaxFiles {
		return nil, ErrRecipeImportTooManyFiles
	}

	out := make([]recipe.ImportCandidate, 0, len(files))
	for _, f := range files {
		out = append(out, s.previewImportFile(f))
	}

	existing, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	nameCount := make(map[string]int, len(existing)+len(out))
	for _, r := range existing {
		nameCount[strings.TrimSpace(r.Name)]++
	}
	// 問題のある候補は名前を信用できないので、重複の判定に加えない。
	for _, c := range out {
		if c.Issue == recipe.ImportIssueNone {
			nameCount[c.Recipe.Name]++
		}
	}
	for i := range out {
		if out[i].Issue == recipe.ImportIssueNone && nameCount[out[i].Recipe.Name] > 1 {
			out[i].Duplicate = true
		}
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].FileName < out[j].FileName })
	return out, nil
}

func (s *RecipeService) previewImportFile(f recipe.ImportFile) recipe.ImportCandidate {
	c := recipe.ImportCandidate{FileName: f.Name}
	if len(f.Content) > recipeImportMaxBytes {
		c.Issue = recipe.ImportIssueTooLarge
		return c
	}
	cfg := s.cfg.current()
	parsed, issue, arg := recipe.ParseImportFile(f.Content, cfg.RecipeImportDefaultServings, cfg.MeasuringUnits, cfg.UnquantifiedWords)
	if issue != recipe.ImportIssueNone {
		c.Issue, c.IssueArg = issue, arg
		return c
	}
	normalized, err := validateRecipeInput(parsed)
	if err != nil {
		// 一覧でレシピ名と材料数を見せるため、読めた値は残す。
		c.Recipe = parsed
		c.Recipe.IngredientCount = len(parsed.Ingredients)
		c.Issue, c.IssueArg = importIssueFromErr(err, parsed)
		return c
	}
	normalized.IngredientCount = len(normalized.Ingredients)
	c.Recipe = normalized
	return c
}

func importIssueFromErr(err error, parsed recipe.Recipe) (recipe.ImportIssue, string) {
	for _, m := range recipeImportIssueByErr {
		if !errors.Is(err, m.err) {
			continue
		}
		if m.issue == recipe.ImportIssueIngredientNameEmpty {
			for i, ing := range parsed.Ingredients {
				if strings.TrimSpace(ing.Name) == "" {
					return m.issue, strconv.Itoa(i + 1)
				}
			}
		}
		if m.issue == recipe.ImportIssueAmountInvalid {
			for _, ing := range parsed.Ingredients {
				if !recipeAmountInRange(ing.Amount) {
					return m.issue, strings.TrimSpace(ing.Name)
				}
			}
		}
		return m.issue, ""
	}
	// 対応表に無いエラーは validateRecipeInput に制約を足したときの取りこぼし。形式の誤りとして見せる。
	return recipe.ImportIssueInvalidYAML, ""
}

// Import は選ばれたレシピをまとめて登録する。フロントから来た値なので全件を検証し直し、1 件でも違反があれば何も書かない。
func (s *RecipeService) Import(ctx context.Context, inputs []recipe.Recipe) ([]recipe.Recipe, error) {
	normalized := make([]recipe.Recipe, 0, len(inputs))
	for _, in := range inputs {
		n, err := validateRecipeInput(in)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, n)
	}
	return s.repo.CreateMany(ctx, normalized)
}

// ExportToFolder は削除済みを除く全レシピを、選ばれたフォルダの中に新しく作るフォルダへ書き出す (add-recipe-export D2)。
// 0 件で空のフォルダを作らないよう、ダイアログより先に DB を読む。
func (s *RecipeService) ExportToFolder(ctx context.Context, dialogTitle string) (recipe.ExportResult, error) {
	recipes, err := s.repo.ListDetailed(ctx)
	if err != nil {
		return recipe.ExportResult{}, err
	}
	if len(recipes) == 0 {
		return recipe.ExportResult{}, ErrRecipeExportEmpty
	}
	if s.picker == nil {
		return recipe.ExportResult{}, fmt.Errorf("%w: no folder picker", ErrRecipeExportWrite)
	}
	parent, err := s.picker.PickFolder(dialogTitle)
	if err != nil {
		return recipe.ExportResult{}, fmt.Errorf("%w: %v", ErrRecipeExportWrite, err)
	}
	if parent == "" {
		return recipe.ExportResult{Cancelled: true}, nil
	}

	dir, err := makeExportDir(parent, s.now())
	if err != nil {
		return recipe.ExportResult{}, fmt.Errorf("%w: %v", ErrRecipeExportWrite, err)
	}
	used := make(map[string]bool, len(recipes))
	for _, r := range recipes {
		path := filepath.Join(dir, recipe.ExportFileName(r.Name, used))
		if err := os.WriteFile(path, []byte(recipe.FormatExportFile(r)), 0o644); err != nil {
			// 作りかけのフォルダを残すと、読み戻したときに一部のレシピだけが入る。
			_ = os.RemoveAll(dir)
			return recipe.ExportResult{}, fmt.Errorf("%w: %v", ErrRecipeExportWrite, err)
		}
	}
	return recipe.ExportResult{Dir: dir, Count: len(recipes)}, nil
}

// 既存のファイルを壊さないよう、毎回新しいフォルダを作る。同じ秒に 2 回書き出したときは番号を付ける (add-recipe-export D5)。
func makeExportDir(parent string, now time.Time) (string, error) {
	parent, err := filepath.Abs(parent)
	if err != nil {
		return "", err
	}
	base := filepath.Join(parent, "meal-cycle-recipes-"+now.Format("20060102-150405"))
	dir := base
	for n := 2; ; n++ {
		err := os.Mkdir(dir, 0o755)
		if err == nil {
			return dir, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", err
		}
		dir = base + "-" + strconv.Itoa(n)
	}
}

// validateRecipeInput は入力を trim (name / ingredient.Name / tags / unit) しつつ
// 制約 (design.md D12 の表) を検証する。空タグは黙って捨てる。
func validateRecipeInput(input recipe.Recipe) (recipe.Recipe, error) {
	out := input
	out.Name = strings.TrimSpace(input.Name)
	if out.Name == "" {
		return recipe.Recipe{}, sqlite.ErrRecipeNameEmpty
	}
	if runeLen(out.Name) > recipeNameMaxRunes {
		return recipe.Recipe{}, ErrRecipeNameTooLong
	}
	if out.Servings < recipeServingsMin || out.Servings > recipeServingsMax {
		return recipe.Recipe{}, fmt.Errorf("%w: got %d", ErrRecipeServingsOutOfRange, out.Servings)
	}
	if runeLen(out.Steps) > recipeStepsMaxRunes {
		return recipe.Recipe{}, ErrRecipeStepsTooLong
	}
	if runeLen(out.Notes) > recipeNotesMaxRunes {
		return recipe.Recipe{}, ErrRecipeNotesTooLong
	}

	trimmedTags := make([]string, 0, len(input.Tags))
	for _, tag := range input.Tags {
		t := strings.TrimSpace(tag)
		if t == "" {
			continue
		}
		if runeLen(t) > recipeTagMaxRunes {
			return recipe.Recipe{}, ErrRecipeTagTooLong
		}
		trimmedTags = append(trimmedTags, t)
	}
	out.Tags = trimmedTags

	trimmedIngs := make([]recipe.RecipeIngredient, 0, len(input.Ingredients))
	for _, ing := range input.Ingredients {
		name := strings.TrimSpace(ing.Name)
		if name == "" {
			return recipe.Recipe{}, ErrRecipeIngredientNameEmpty
		}
		if !recipeAmountInRange(ing.Amount) {
			return recipe.Recipe{}, fmt.Errorf("%w: got %v", ErrRecipeIngredientAmountOutOfRange, ing.Amount)
		}
		trimmedIngs = append(trimmedIngs, recipe.RecipeIngredient{
			Name:   name,
			Amount: ing.Amount,
			Unit:   strings.TrimSpace(ing.Unit),
		})
	}
	out.Ingredients = trimmedIngs

	return out, nil
}

// 量 0 は「量を持たない材料」(塩 少々 など) を表すので下限に含める。
func recipeAmountInRange(amount float64) bool {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return false
	}
	return amount >= 0 && amount <= recipeAmountMax
}

// rune 数を返す。byte ではなく文字数で制約を判定するため。
func runeLen(s string) int {
	return len([]rune(s))
}
