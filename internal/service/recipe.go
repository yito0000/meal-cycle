package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
)

// 入力バリデーションエラー。フロントで i18n key に対応付ける (design.md D12)。
var (
	ErrRecipeNameTooLong         = errors.New("recipe name too long")
	ErrRecipeServingsOutOfRange  = errors.New("recipe servings out of range")
	ErrRecipeIngredientNameEmpty = errors.New("recipe ingredient name empty")
	ErrRecipeStepsTooLong        = errors.New("recipe steps too long")
	ErrRecipeNotesTooLong        = errors.New("recipe notes too long")
	ErrRecipeTagTooLong          = errors.New("recipe tag too long")
)

// RecipeService はレシピの List / GetByID / Create / Update / Delete をフロントに公開する。
// Create / Update は保存前に validateRecipeInput を走らせ、違反時は Repository
// に到達せず型付きエラーを返す。
type RecipeService struct {
	repo *sqlite.RecipeRepo
}

func NewRecipeService(db *sql.DB) *RecipeService {
	return &RecipeService{repo: sqlite.NewRecipeRepo(db)}
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
		trimmedIngs = append(trimmedIngs, recipe.RecipeIngredient{
			Name:   name,
			Amount: ing.Amount,
			Unit:   strings.TrimSpace(ing.Unit),
		})
	}
	out.Ingredients = trimmedIngs

	return out, nil
}

// rune 数を返す。byte ではなく文字数で制約を判定するため。
func runeLen(s string) int {
	return len([]rune(s))
}
