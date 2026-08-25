package main

import (
	"context"

	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

// RecipeService は S02 レシピ一覧のデータをフロントエンドに公開する Wails
// サービス。
//
// 本 proposal (add-recipe-listing) では実装を固定 in-memory データセット
// に留め、UI / Wails バインディング / ドメイン型を先に確定させる。
// リポジトリ実装への差し替えは後続の add-recipe-crud で行う。
type RecipeService struct{}

// NewRecipeService は RecipeService を生成する。
func NewRecipeService() *RecipeService {
	return &RecipeService{}
}

// List はサービスが把握している全レシピを返す。リポジトリが入るまでは
// 固定データセットを返し、フロント側で「タグ複数」「タグ 0 件」の描画
// パスを両方確認できるようにしてある。
//
// 呼び出し側 (テストの JSON エンコーダ含む) が内部 fixture を書き換えて
// しまわないよう、毎回新しいスライスを返す。
func (s *RecipeService) List(_ context.Context) ([]recipe.Recipe, error) {
	out := make([]recipe.Recipe, len(fixtureRecipes))
	copy(out, fixtureRecipes)
	return out, nil
}

// fixtureRecipes は List が返すプレースホルダデータ。日付は time.Now
// 相対ではなく固定文字列にして、テストの再現性を確保している。
// タグ列の描画確認のため、意図的に「タグ 2 個以上のレシピ」と「タグ 0 個
// のレシピ」をそれぞれ最低 1 件含める。
var fixtureRecipes = []recipe.Recipe{
	{
		ID:              "r-001",
		Name:            "肉じゃが",
		Tags:            []string{"和食", "煮物"},
		IngredientCount: 7,
		UpdatedAt:       "2026-08-20",
	},
	{
		ID:              "r-002",
		Name:            "麻婆豆腐",
		Tags:            []string{"中華", "辛い", "簡単"},
		IngredientCount: 8,
		UpdatedAt:       "2026-08-18",
	},
	{
		ID:              "r-003",
		Name:            "鮭のホイル焼き",
		Tags:            []string{"魚"},
		IngredientCount: 5,
		UpdatedAt:       "2026-08-15",
	},
	{
		ID:              "r-004",
		Name:            "トースト",
		Tags:            []string{},
		IngredientCount: 2,
		UpdatedAt:       "2026-08-10",
	},
	{
		ID:              "r-005",
		Name:            "ほうれん草の胡麻和え",
		Tags:            []string{"和食", "副菜"},
		IngredientCount: 4,
		UpdatedAt:       "2026-08-05",
	},
	{
		ID:              "r-006",
		Name:            "焼きうどん",
		Tags:            []string{"麺"},
		IngredientCount: 6,
		UpdatedAt:       "2026-07-30",
	},
}
