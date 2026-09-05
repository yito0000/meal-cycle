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

// バリデーション上限。将来 S09 設定画面で可変にする可能性があるが、
// ユーザーが変えたい値ではなく実装上限なので defaults.go には置かない
// (design.md D12 参照)。
const (
	recipeNameMaxRunes  = 200
	recipeServingsMin   = 1
	recipeServingsMax   = 99
	recipeStepsMaxRunes = 4000
	recipeNotesMaxRunes = 2000
	recipeTagMaxRunes   = 30
)

// 入力バリデーションエラー。フロントで i18n key に対応付ける (design.md D12 の表参照)。
var (
	ErrRecipeNameTooLong         = errors.New("recipe name too long")
	ErrRecipeServingsOutOfRange  = errors.New("recipe servings out of range")
	ErrRecipeIngredientNameEmpty = errors.New("recipe ingredient name empty")
	ErrRecipeStepsTooLong        = errors.New("recipe steps too long")
	ErrRecipeNotesTooLong        = errors.New("recipe notes too long")
	ErrRecipeTagTooLong          = errors.New("recipe tag too long")
)

// RecipeService はレシピの一覧取得 / 詳細取得 / 新規作成 / 更新をフロント
// エンドに公開する Wails サービス。
//
// 本 proposal (add-recipe-edit) で前 proposal の固定 in-memory fixture を
// SQLite 実装に差し替え、Create / Update / GetByID を追加した。削除は
// 別 proposal (add-recipe-deletion) の担当。
//
// Create / Update は保存前に入力バリデーション (`validateRecipeInput`) を
// 走らせ、違反時は型付きエラーで返す (フロントで i18n key に対応付ける)。
type RecipeService struct {
	repo *sqlite.RecipeRepo
}

// NewRecipeService は RecipeService を db に紐付けて生成する。
func NewRecipeService(db *sql.DB) *RecipeService {
	return &RecipeService{repo: sqlite.NewRecipeRepo(db)}
}

// List は登録済みの全レシピを updated_at 降順で返す。S02 一覧向けの
// projection のため Ingredients / Steps / Notes は空値で返し、
// IngredientCount のみ集計する (詳細は Repository.List を参照)。
func (s *RecipeService) List(ctx context.Context) ([]recipe.Recipe, error) {
	return s.repo.List(ctx)
}

// GetByID は指定 ID のレシピを詳細フィールド込みで返す。
// 該当なしは sqlite.ErrRecipeNotFound を返す。
func (s *RecipeService) GetByID(ctx context.Context, id string) (recipe.Recipe, error) {
	return s.repo.GetByID(ctx, id)
}

// Create は input からレシピを新規作成する。
//
// input.ID / input.CreatedAt / input.UpdatedAt / input.IngredientCount は
// Repository 側で決定するため入力としては無視される。呼び出し側 (S04)
// は name / servings / tags / ingredients / steps / notes を詰めるだけで
// よい。バリデーション違反時は Repository に到達せず即エラーを返す。
func (s *RecipeService) Create(ctx context.Context, input recipe.Recipe) (recipe.Recipe, error) {
	normalized, err := validateRecipeInput(input)
	if err != nil {
		return recipe.Recipe{}, err
	}
	return s.repo.Create(ctx, normalized)
}

// Update は指定 ID のレシピを input で上書きする。updated_at は現在時刻に
// 更新される。input.ID / input.CreatedAt / input.UpdatedAt /
// input.IngredientCount は無視される (Create と同様)。
func (s *RecipeService) Update(ctx context.Context, id string, input recipe.Recipe) (recipe.Recipe, error) {
	normalized, err := validateRecipeInput(input)
	if err != nil {
		return recipe.Recipe{}, err
	}
	return s.repo.Update(ctx, id, normalized)
}

// validateRecipeInput は S04 から受け取った Recipe を検証し、trim 済み
// (name / ingredient.name / tags) の Recipe を返す。違反時は該当エラー。
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

	// Tags: 各要素を trim。空タグは黙って捨てる。長さチェック。
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

	// Ingredients: 各行の name を trim して必須チェック。unit も trim。
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

// runeLen は UTF-8 rune 数を返す (byte 数ではなく)。
func runeLen(s string) int {
	return len([]rune(s))
}
