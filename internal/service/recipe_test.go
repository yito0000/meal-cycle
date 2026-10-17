package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/yito0000/meal-cycle/internal/config"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

func newRecipeServiceForTest(t *testing.T) *RecipeService {
	t.Helper()
	return newRecipeServiceWithPicker(t, nil)
}

func newRecipeServiceWithPicker(t *testing.T, picker FolderPicker) *RecipeService {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := sqlite.Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewRecipeService(db, staticConfig(config.Defaults()), picker)
}

// validRecipe は各テストで使う「バリデーションを通る最小の入力」。
// 個別テストは変更したいフィールドだけ上書きして使う。
func validRecipe(name string) recipe.Recipe {
	return recipe.Recipe{Name: name, Servings: 1}
}

func TestRecipeService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("should return an empty list when the database is fresh", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		got, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("want empty, got %d", len(got))
		}
	})

	t.Run("should return recipes ordered by updatedAt desc when multiple exist", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		for _, name := range []string{"a", "b", "c"} {
			if _, err := svc.Create(ctx, validRecipe(name)); err != nil {
				t.Fatalf("Create %s: %v", name, err)
			}
		}
		got, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("want 3, got %d", len(got))
		}
		for i := 1; i < len(got); i++ {
			if got[i-1].UpdatedAt < got[i].UpdatedAt {
				t.Errorf("not sorted desc at %d: %q vs %q",
					i, got[i-1].UpdatedAt, got[i].UpdatedAt)
			}
		}
	})
}

func TestRecipeService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("should create a recipe and make it visible in List when Create succeeds", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		if _, err := svc.Create(ctx, validRecipe("肉じゃが")); err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 1 || got[0].Name != "肉じゃが" {
			t.Errorf("unexpected list: %+v", got)
		}
	})

	t.Run("should return the created recipe with a non-empty ID and updatedAt", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		got, err := svc.Create(ctx, validRecipe("肉じゃが"))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if got.ID == "" || got.UpdatedAt == "" {
			t.Errorf("want non-empty ID and updatedAt, got %+v", got)
		}
	})

	t.Run("should return ErrRecipeNameEmpty when Create is called with an empty name", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		_, err := svc.Create(ctx, recipe.Recipe{Name: "", Servings: 1})
		if !errors.Is(err, sqlite.ErrRecipeNameEmpty) {
			t.Errorf("want ErrRecipeNameEmpty, got %v", err)
		}
	})

	t.Run("should return ErrRecipeNameEmpty when Create is called with whitespace-only name", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		_, err := svc.Create(ctx, recipe.Recipe{Name: "   ", Servings: 1})
		if !errors.Is(err, sqlite.ErrRecipeNameEmpty) {
			t.Errorf("want ErrRecipeNameEmpty, got %v", err)
		}
	})

	t.Run("should return ErrRecipeNameTooLong when Create is called with a 201-character name", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		_, err := svc.Create(ctx, recipe.Recipe{Name: strings.Repeat("あ", 201), Servings: 1})
		if !errors.Is(err, ErrRecipeNameTooLong) {
			t.Errorf("want ErrRecipeNameTooLong, got %v", err)
		}
	})

	t.Run("should return ErrRecipeServingsOutOfRange when Create is called with servings=0", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		_, err := svc.Create(ctx, recipe.Recipe{Name: "x", Servings: 0})
		if !errors.Is(err, ErrRecipeServingsOutOfRange) {
			t.Errorf("want ErrRecipeServingsOutOfRange, got %v", err)
		}
	})

	t.Run("should return ErrRecipeServingsOutOfRange when Create is called with servings=100", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		_, err := svc.Create(ctx, recipe.Recipe{Name: "x", Servings: 100})
		if !errors.Is(err, ErrRecipeServingsOutOfRange) {
			t.Errorf("want ErrRecipeServingsOutOfRange, got %v", err)
		}
	})

	t.Run("should return ErrRecipeIngredientNameEmpty when an ingredient row has empty Name", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		_, err := svc.Create(ctx, recipe.Recipe{
			Name: "x", Servings: 1,
			Ingredients: []recipe.RecipeIngredient{{Name: "  ", Amount: 1, Unit: "個"}},
		})
		if !errors.Is(err, ErrRecipeIngredientNameEmpty) {
			t.Errorf("want ErrRecipeIngredientNameEmpty, got %v", err)
		}
	})

	t.Run("should reject an ingredient with a negative amount", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		in := validRecipe("x")
		in.Ingredients = []recipe.RecipeIngredient{{Name: "玉ねぎ", Amount: -1}}
		_, err := svc.Create(ctx, in)
		if !errors.Is(err, ErrRecipeIngredientAmountOutOfRange) {
			t.Errorf("want ErrRecipeIngredientAmountOutOfRange, got %v", err)
		}
		list, _ := svc.List(ctx)
		if len(list) != 0 {
			t.Errorf("want no recipes, got %d", len(list))
		}
	})

	t.Run("should reject an ingredient whose amount exceeds 99999", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		in := validRecipe("x")
		in.Ingredients = []recipe.RecipeIngredient{{Name: "玉ねぎ", Amount: 100000}}
		_, err := svc.Create(ctx, in)
		if !errors.Is(err, ErrRecipeIngredientAmountOutOfRange) {
			t.Errorf("want ErrRecipeIngredientAmountOutOfRange, got %v", err)
		}
	})

	t.Run("should accept an ingredient whose amount is 0", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		in := validRecipe("x")
		in.Ingredients = []recipe.RecipeIngredient{{Name: "塩", Amount: 0}}
		created, err := svc.Create(ctx, in)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if created.Ingredients[0].Amount != 0 {
			t.Errorf("want amount 0, got %v", created.Ingredients[0].Amount)
		}
	})

	t.Run("should accept an ingredient whose amount is exactly 99999", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		in := validRecipe("x")
		in.Ingredients = []recipe.RecipeIngredient{{Name: "水", Amount: 99999, Unit: "ml"}}
		created, err := svc.Create(ctx, in)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if created.Ingredients[0].Amount != 99999 {
			t.Errorf("want amount 99999, got %v", created.Ingredients[0].Amount)
		}
	})

	t.Run("should return ErrRecipeStepsTooLong when Steps exceeds 4000 characters", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		_, err := svc.Create(ctx, recipe.Recipe{
			Name: "x", Servings: 1, Steps: strings.Repeat("あ", 4001),
		})
		if !errors.Is(err, ErrRecipeStepsTooLong) {
			t.Errorf("want ErrRecipeStepsTooLong, got %v", err)
		}
	})

	t.Run("should return ErrRecipeNotesTooLong when Notes exceeds 2000 characters", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		_, err := svc.Create(ctx, recipe.Recipe{
			Name: "x", Servings: 1, Notes: strings.Repeat("あ", 2001),
		})
		if !errors.Is(err, ErrRecipeNotesTooLong) {
			t.Errorf("want ErrRecipeNotesTooLong, got %v", err)
		}
	})

	t.Run("should return ErrRecipeTagTooLong when a single tag exceeds 30 characters", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		_, err := svc.Create(ctx, recipe.Recipe{
			Name: "x", Servings: 1, Tags: []string{strings.Repeat("あ", 31)},
		})
		if !errors.Is(err, ErrRecipeTagTooLong) {
			t.Errorf("want ErrRecipeTagTooLong, got %v", err)
		}
	})

	t.Run("should silently drop empty tags after trim", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		created, err := svc.Create(ctx, recipe.Recipe{
			Name: "x", Servings: 1, Tags: []string{"和食", "   ", "煮物"},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if len(created.Tags) != 2 || created.Tags[0] != "和食" || created.Tags[1] != "煮物" {
			t.Errorf("unexpected tags: %v", created.Tags)
		}
	})
}

func TestRecipeService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("should update an existing recipe when Update is called with valid input", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		created, err := svc.Create(ctx, validRecipe("old"))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		updated, err := svc.Update(ctx, created.ID, validRecipe("new"))
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.Name != "new" {
			t.Errorf("want new, got %q", updated.Name)
		}
	})

	t.Run("should refresh updatedAt when Update is called", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		created, err := svc.Create(ctx, validRecipe("x"))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		updated, err := svc.Update(ctx, created.ID, validRecipe("x"))
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.UpdatedAt <= created.UpdatedAt {
			t.Errorf("updatedAt not advanced: %q <= %q", updated.UpdatedAt, created.UpdatedAt)
		}
	})

	t.Run("should replace ingredients when Update is called with a different ingredient set", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		created, err := svc.Create(ctx, recipe.Recipe{
			Name: "x", Servings: 1,
			Ingredients: []recipe.RecipeIngredient{{Name: "玉ねぎ", Amount: 1, Unit: "個"}},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := svc.Update(ctx, created.ID, recipe.Recipe{
			Name: "x", Servings: 1,
			Ingredients: []recipe.RecipeIngredient{{Name: "にんじん", Amount: 2, Unit: "本"}},
		}); err != nil {
			t.Fatalf("Update: %v", err)
		}
		got, err := svc.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if len(got.Ingredients) != 1 || got.Ingredients[0].Name != "にんじん" {
			t.Errorf("ingredients not replaced: %+v", got.Ingredients)
		}
	})

	t.Run("should leave the recipe unchanged when an update has an out-of-range amount", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		in := validRecipe("x")
		in.Ingredients = []recipe.RecipeIngredient{{Name: "玉ねぎ", Amount: 1, Unit: "個"}}
		created, err := svc.Create(ctx, in)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		bad := validRecipe("x")
		bad.Ingredients = []recipe.RecipeIngredient{{Name: "玉ねぎ", Amount: 100000, Unit: "個"}}
		if _, err := svc.Update(ctx, created.ID, bad); !errors.Is(err, ErrRecipeIngredientAmountOutOfRange) {
			t.Fatalf("want ErrRecipeIngredientAmountOutOfRange, got %v", err)
		}
		got, err := svc.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if len(got.Ingredients) != 1 || got.Ingredients[0].Amount != 1 {
			t.Errorf("ingredients changed: %+v", got.Ingredients)
		}
		if got.UpdatedAt != created.UpdatedAt {
			t.Errorf("updatedAt changed: %q -> %q", created.UpdatedAt, got.UpdatedAt)
		}
	})

	t.Run("should return ErrRecipeNotFound when Update is called with an unknown id", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		_, err := svc.Update(ctx, "no-such-id", validRecipe("x"))
		if !errors.Is(err, sqlite.ErrRecipeNotFound) {
			t.Errorf("want ErrRecipeNotFound, got %v", err)
		}
	})

	t.Run("should return ErrRecipeArchived when the recipe is deleted", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		created, err := svc.Create(ctx, validRecipe("肉じゃが"))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := svc.Delete(ctx, created.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		_, err = svc.Update(ctx, created.ID, validRecipe("豚汁"))
		if !errors.Is(err, sqlite.ErrRecipeArchived) {
			t.Errorf("want ErrRecipeArchived, got %v", err)
		}
	})
}

func TestRecipeService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("should return the recipe when GetByID is called with an existing id", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		created, err := svc.Create(ctx, validRecipe("肉じゃが"))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := svc.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Name != "肉じゃが" {
			t.Errorf("want 肉じゃが, got %q", got.Name)
		}
	})

	t.Run("should return ErrRecipeNotFound when GetByID is called with an unknown id", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		_, err := svc.GetByID(ctx, "no-such-id")
		if !errors.Is(err, sqlite.ErrRecipeNotFound) {
			t.Errorf("want ErrRecipeNotFound, got %v", err)
		}
	})

	t.Run("should preserve Markdown text in Steps and Notes across Create -> GetByID", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		markdown := "## 下ごしらえ\n- 玉ねぎを切る\n- 豚肉を漬ける\n"
		created, err := svc.Create(ctx, recipe.Recipe{
			Name: "肉じゃが", Servings: 2, Steps: markdown, Notes: "**メモ**",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := svc.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Steps != markdown {
			t.Errorf("Steps not preserved:\nwant %q\ngot  %q", markdown, got.Steps)
		}
		if got.Notes != "**メモ**" {
			t.Errorf("Notes not preserved: got %q", got.Notes)
		}
	})
}

func TestRecipeService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("should hide the recipe from List when deleted", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		target, err := svc.Create(ctx, validRecipe("肉じゃが"))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := svc.Create(ctx, validRecipe("豚汁")); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := svc.Delete(ctx, target.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		got, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 1 || got[0].Name != "豚汁" {
			t.Errorf("want only 豚汁, got %+v", got)
		}
	})

	t.Run("should keep returning the recipe from GetByID with archivedAt when deleted", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		created, err := svc.Create(ctx, validRecipe("肉じゃが"))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := svc.Delete(ctx, created.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		got, err := svc.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Name != "肉じゃが" || got.ArchivedAt == "" {
			t.Errorf("want 肉じゃが with archivedAt, got %+v", got)
		}
	})

	t.Run("should return ErrRecipeNotFound when id does not exist", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		if err := svc.Delete(ctx, "no-such-id"); !errors.Is(err, sqlite.ErrRecipeNotFound) {
			t.Errorf("want ErrRecipeNotFound, got %v", err)
		}
	})

	t.Run("should succeed without error when the recipe is already deleted", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		created, err := svc.Create(ctx, validRecipe("肉じゃが"))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := svc.Delete(ctx, created.ID); err != nil {
			t.Fatalf("first Delete: %v", err)
		}
		if err := svc.Delete(ctx, created.ID); err != nil {
			t.Errorf("second Delete: want nil, got %v", err)
		}
	})
}

func importFile(name, recipeName string) recipe.ImportFile {
	return recipe.ImportFile{Name: name, Content: "---\nname: \"" + recipeName + "\"\nservings: 1\n---\n## 手順\n煮る\n"}
}

func TestRecipeService_PreviewImport(t *testing.T) {
	ctx := context.Background()

	t.Run("should return candidates sorted by file name when multiple files are given", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		got, err := svc.PreviewImport(ctx, []recipe.ImportFile{importFile("c.md", "C"), importFile("a.md", "A"), importFile("b.md", "B")})
		if err != nil {
			t.Fatalf("PreviewImport: %v", err)
		}
		var names []string
		for _, c := range got {
			names = append(names, c.FileName)
		}
		if strings.Join(names, ",") != "a.md,b.md,c.md" {
			t.Errorf("order: got %v", names)
		}
	})

	t.Run("should fill servings with the default when servings is omitted", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		got, err := svc.PreviewImport(ctx, []recipe.ImportFile{{Name: "a.md", Content: "---\nname: カレー\n---\n"}})
		if err != nil {
			t.Fatalf("PreviewImport: %v", err)
		}
		if got[0].Issue != "" || got[0].Recipe.Servings != svc.cfg.current().RecipeImportDefaultServings {
			t.Errorf("got issue %q servings %d", got[0].Issue, got[0].Recipe.Servings)
		}
	})

	t.Run("should return ErrRecipeImportTooManyFiles when more than 100 files are given", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		files := make([]recipe.ImportFile, recipeImportMaxFiles+1)
		for i := range files {
			files[i] = importFile("x.md", "x")
		}
		if _, err := svc.PreviewImport(ctx, files); !errors.Is(err, ErrRecipeImportTooManyFiles) {
			t.Errorf("want ErrRecipeImportTooManyFiles, got %v", err)
		}
	})

	t.Run("should mark too_large when a file exceeds 64 KB", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		f := importFile("big.md", "big")
		f.Content += strings.Repeat("a", recipeImportMaxBytes)
		got, err := svc.PreviewImport(ctx, []recipe.ImportFile{f})
		if err != nil {
			t.Fatalf("PreviewImport: %v", err)
		}
		if got[0].Issue != recipe.ImportIssueTooLarge {
			t.Errorf("issue: got %q", got[0].Issue)
		}
	})

	t.Run("should mark name_empty when the name is missing", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		got, err := svc.PreviewImport(ctx, []recipe.ImportFile{{Name: "a.md", Content: "---\nservings: 2\n---\n"}})
		if err != nil {
			t.Fatalf("PreviewImport: %v", err)
		}
		if got[0].Issue != recipe.ImportIssueNameEmpty {
			t.Errorf("issue: got %q", got[0].Issue)
		}
	})

	t.Run("should mark steps_too_long when the steps exceed 4000 characters", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		content := "---\nname: x\n---\n" + strings.Repeat("あ", recipeStepsMaxRunes+1)
		got, err := svc.PreviewImport(ctx, []recipe.ImportFile{{Name: "a.md", Content: content}})
		if err != nil {
			t.Fatalf("PreviewImport: %v", err)
		}
		if got[0].Issue != recipe.ImportIssueStepsTooLong {
			t.Errorf("issue: got %q", got[0].Issue)
		}
	})

	t.Run("should mark ingredient_name_empty with the ingredient position when an ingredient has no name", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		content := "---\nname: x\ningredients:\n  - name: 玉ねぎ\n    amount: 1\n  - amount: 2\n---\n"
		got, err := svc.PreviewImport(ctx, []recipe.ImportFile{{Name: "a.md", Content: content}})
		if err != nil {
			t.Fatalf("PreviewImport: %v", err)
		}
		if got[0].Issue != recipe.ImportIssueIngredientNameEmpty || got[0].IssueArg != "2" {
			t.Errorf("got issue %q arg %q", got[0].Issue, got[0].IssueArg)
		}
	})

	t.Run("should report amount_invalid with the ingredient name when a file has an amount over 99999", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		content := "---\nname: x\ningredients:\n  - name: 玉ねぎ\n    amount: 1\n  - name: 水\n    amount: 100000\n---\n"
		got, err := svc.PreviewImport(ctx, []recipe.ImportFile{{Name: "a.md", Content: content}})
		if err != nil {
			t.Fatalf("PreviewImport: %v", err)
		}
		if got[0].Issue != recipe.ImportIssueAmountInvalid || got[0].IssueArg != "水" {
			t.Errorf("got issue %q arg %q", got[0].Issue, got[0].IssueArg)
		}
	})

	t.Run("should mark duplicate when the name matches an existing recipe", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		if _, err := svc.Create(ctx, validRecipe("肉じゃが")); err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := svc.PreviewImport(ctx, []recipe.ImportFile{importFile("a.md", "肉じゃが"), importFile("b.md", "カレー")})
		if err != nil {
			t.Fatalf("PreviewImport: %v", err)
		}
		if !got[0].Duplicate || got[1].Duplicate {
			t.Errorf("duplicate: got %v, %v", got[0].Duplicate, got[1].Duplicate)
		}
	})

	t.Run("should not mark duplicate when the name matches only an archived recipe", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		created, err := svc.Create(ctx, validRecipe("肉じゃが"))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := svc.Delete(ctx, created.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		got, err := svc.PreviewImport(ctx, []recipe.ImportFile{importFile("a.md", "肉じゃが")})
		if err != nil {
			t.Fatalf("PreviewImport: %v", err)
		}
		if got[0].Duplicate {
			t.Error("want not duplicate")
		}
	})

	t.Run("should mark both candidates duplicate when two files share a name", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		got, err := svc.PreviewImport(ctx, []recipe.ImportFile{importFile("a.md", "カレー"), importFile("b.md", " カレー ")})
		if err != nil {
			t.Fatalf("PreviewImport: %v", err)
		}
		if !got[0].Duplicate || !got[1].Duplicate {
			t.Errorf("duplicate: got %v, %v", got[0].Duplicate, got[1].Duplicate)
		}
	})

	t.Run("should not mark duplicate against a candidate that has an issue", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		bad := recipe.ImportFile{Name: "b.md", Content: "---\nname: カレー\nservings: 0\n---\n"}
		got, err := svc.PreviewImport(ctx, []recipe.ImportFile{importFile("a.md", "カレー"), bad})
		if err != nil {
			t.Fatalf("PreviewImport: %v", err)
		}
		if got[0].Duplicate {
			t.Error("want a.md not duplicate")
		}
		if got[1].Issue != recipe.ImportIssueServingsOutOfRange {
			t.Errorf("b.md issue: got %q", got[1].Issue)
		}
	})

	t.Run("should not write anything to the database when previewing", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		if _, err := svc.PreviewImport(ctx, []recipe.ImportFile{importFile("a.md", "カレー")}); err != nil {
			t.Fatalf("PreviewImport: %v", err)
		}
		list, _ := svc.List(ctx)
		if len(list) != 0 {
			t.Errorf("want no recipes, got %d", len(list))
		}
	})
}

func TestRecipeService_Import(t *testing.T) {
	ctx := context.Background()

	t.Run("should create all recipes when every input is valid", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		created, err := svc.Import(ctx, []recipe.Recipe{validRecipe("肉じゃが"), validRecipe("カレー")})
		if err != nil {
			t.Fatalf("Import: %v", err)
		}
		list, _ := svc.List(ctx)
		if len(created) != 2 || len(list) != 2 {
			t.Errorf("want 2 created and listed, got %d / %d", len(created), len(list))
		}
	})

	t.Run("should create a separate recipe when the name matches an existing recipe", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		original := validRecipe("肉じゃが")
		original.Notes = "元の内容"
		existing, err := svc.Create(ctx, original)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := svc.Import(ctx, []recipe.Recipe{validRecipe("肉じゃが")}); err != nil {
			t.Fatalf("Import: %v", err)
		}
		list, _ := svc.List(ctx)
		if len(list) != 2 {
			t.Errorf("want 2 recipes, got %d", len(list))
		}
		got, _ := svc.GetByID(ctx, existing.ID)
		if got.Notes != "元の内容" {
			t.Errorf("existing notes changed: %q", got.Notes)
		}
	})

	t.Run("should create nothing when one input violates a constraint", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		bad := validRecipe("カレー")
		bad.Servings = 0
		_, err := svc.Import(ctx, []recipe.Recipe{validRecipe("肉じゃが"), bad})
		if !errors.Is(err, ErrRecipeServingsOutOfRange) {
			t.Fatalf("want ErrRecipeServingsOutOfRange, got %v", err)
		}
		list, _ := svc.List(ctx)
		if len(list) != 0 {
			t.Errorf("want no recipes, got %d", len(list))
		}
	})

	t.Run("should reject the whole import when one recipe has a negative amount", func(t *testing.T) {
		svc := newRecipeServiceForTest(t)
		bad := validRecipe("カレー")
		bad.Ingredients = []recipe.RecipeIngredient{{Name: "玉ねぎ", Amount: -1}}
		_, err := svc.Import(ctx, []recipe.Recipe{validRecipe("肉じゃが"), bad})
		if !errors.Is(err, ErrRecipeIngredientAmountOutOfRange) {
			t.Fatalf("want ErrRecipeIngredientAmountOutOfRange, got %v", err)
		}
		list, _ := svc.List(ctx)
		if len(list) != 0 {
			t.Errorf("want no recipes, got %d", len(list))
		}
	})
}

type stubFolderPicker struct {
	dir   string
	err   error
	calls int
}

func (p *stubFolderPicker) PickFolder(string) (string, error) {
	p.calls++
	return p.dir, p.err
}

func exportedFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		out[e.Name()] = string(b)
	}
	return out
}

func TestRecipeService_ExportToFolder(t *testing.T) {
	ctx := context.Background()

	create := func(t *testing.T, svc *RecipeService, r recipe.Recipe) recipe.Recipe {
		t.Helper()
		created, err := svc.Create(ctx, r)
		if err != nil {
			t.Fatalf("Create %q: %v", r.Name, err)
		}
		return created
	}

	t.Run("should write every active recipe as one file when a folder is picked", func(t *testing.T) {
		picked := t.TempDir()
		svc := newRecipeServiceWithPicker(t, &stubFolderPicker{dir: picked})
		create(t, svc, validRecipe("肉じゃが"))
		create(t, svc, validRecipe("味噌汁"))

		got, err := svc.ExportToFolder(ctx, "title")
		if err != nil {
			t.Fatalf("ExportToFolder: %v", err)
		}
		if got.Cancelled || got.Count != 2 {
			t.Errorf("result: %+v", got)
		}
		if filepath.Dir(got.Dir) != picked || !strings.HasPrefix(filepath.Base(got.Dir), "meal-cycle-recipes-") {
			t.Errorf("dir %q is not a new folder under %q", got.Dir, picked)
		}
		files := exportedFiles(t, got.Dir)
		_, first := files["肉じゃが.md"]
		_, second := files["味噌汁.md"]
		if !first || !second || len(files) != 2 {
			t.Errorf("files: %v", files)
		}
	})

	t.Run("should skip archived recipes when some are deleted", func(t *testing.T) {
		svc := newRecipeServiceWithPicker(t, &stubFolderPicker{dir: t.TempDir()})
		create(t, svc, validRecipe("肉じゃが"))
		archived := create(t, svc, validRecipe("味噌汁"))
		if err := svc.Delete(ctx, archived.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}

		got, err := svc.ExportToFolder(ctx, "title")
		if err != nil {
			t.Fatalf("ExportToFolder: %v", err)
		}
		files := exportedFiles(t, got.Dir)
		if _, ok := files["肉じゃが.md"]; !ok || len(files) != 1 || got.Count != 1 {
			t.Errorf("files: %v (count %d)", files, got.Count)
		}
	})

	t.Run("should write nothing and report cancelled when the picker returns empty", func(t *testing.T) {
		svc := newRecipeServiceWithPicker(t, &stubFolderPicker{dir: ""})
		create(t, svc, validRecipe("肉じゃが"))

		got, err := svc.ExportToFolder(ctx, "title")
		if err != nil {
			t.Fatalf("ExportToFolder: %v", err)
		}
		if !got.Cancelled || got.Dir != "" || got.Count != 0 {
			t.Errorf("result: %+v", got)
		}
	})

	t.Run("should return ErrRecipeExportWrite and leave the picked folder unchanged when it is not writable", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root ignores directory permissions")
		}
		picked := t.TempDir()
		if err := os.WriteFile(filepath.Join(picked, "keep.txt"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(picked, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(picked, 0o755) })
		svc := newRecipeServiceWithPicker(t, &stubFolderPicker{dir: picked})
		create(t, svc, validRecipe("肉じゃが"))

		_, err := svc.ExportToFolder(ctx, "title")
		if !errors.Is(err, ErrRecipeExportWrite) {
			t.Fatalf("want ErrRecipeExportWrite, got %v", err)
		}
		files := exportedFiles(t, picked)
		if len(files) != 1 || files["keep.txt"] != "x" {
			t.Errorf("picked folder changed: %v", files)
		}
	})

	t.Run("should return ErrRecipeExportWrite when the picker fails", func(t *testing.T) {
		svc := newRecipeServiceWithPicker(t, &stubFolderPicker{err: errors.New("dialog broken")})
		create(t, svc, validRecipe("肉じゃが"))

		if _, err := svc.ExportToFolder(ctx, "title"); !errors.Is(err, ErrRecipeExportWrite) {
			t.Fatalf("want ErrRecipeExportWrite, got %v", err)
		}
	})

	t.Run("should return ErrRecipeExportEmpty without opening the picker when no recipes exist", func(t *testing.T) {
		picker := &stubFolderPicker{dir: t.TempDir()}
		svc := newRecipeServiceWithPicker(t, picker)

		if _, err := svc.ExportToFolder(ctx, "title"); !errors.Is(err, ErrRecipeExportEmpty) {
			t.Fatalf("want ErrRecipeExportEmpty, got %v", err)
		}
		if picker.calls != 0 {
			t.Errorf("picker opened %d times", picker.calls)
		}
	})

	t.Run("should create a suffixed folder when the timestamped folder already exists", func(t *testing.T) {
		picked := t.TempDir()
		svc := newRecipeServiceWithPicker(t, &stubFolderPicker{dir: picked})
		fixed := time.Date(2026, 10, 18, 9, 30, 15, 0, time.Local)
		svc.now = func() time.Time { return fixed }
		if err := os.Mkdir(filepath.Join(picked, "meal-cycle-recipes-20261018-093015"), 0o755); err != nil {
			t.Fatal(err)
		}
		create(t, svc, validRecipe("肉じゃが"))

		got, err := svc.ExportToFolder(ctx, "title")
		if err != nil {
			t.Fatalf("ExportToFolder: %v", err)
		}
		if want := filepath.Join(picked, "meal-cycle-recipes-20261018-093015-2"); got.Dir != want {
			t.Errorf("dir: got %q, want %q", got.Dir, want)
		}
	})

	t.Run("should number files when two recipes share a name", func(t *testing.T) {
		svc := newRecipeServiceWithPicker(t, &stubFolderPicker{dir: t.TempDir()})
		create(t, svc, validRecipe("カレー"))
		create(t, svc, validRecipe("カレー"))

		got, err := svc.ExportToFolder(ctx, "title")
		if err != nil {
			t.Fatalf("ExportToFolder: %v", err)
		}
		files := exportedFiles(t, got.Dir)
		_, first := files["カレー.md"]
		_, second := files["カレー-2.md"]
		if !first || !second || len(files) != 2 {
			t.Errorf("files: %v", files)
		}
	})

	t.Run("should replace characters unusable in file names", func(t *testing.T) {
		svc := newRecipeServiceWithPicker(t, &stubFolderPicker{dir: t.TempDir()})
		create(t, svc, validRecipe("鶏/豚の生姜焼き"))

		got, err := svc.ExportToFolder(ctx, "title")
		if err != nil {
			t.Fatalf("ExportToFolder: %v", err)
		}
		files := exportedFiles(t, got.Dir)
		content, ok := files["鶏_豚の生姜焼き.md"]
		if !ok {
			t.Fatalf("files: %v", files)
		}
		if !strings.Contains(content, "鶏/豚の生姜焼き") {
			t.Errorf("recipe name changed in content:\n%s", content)
		}
	})

	t.Run("should produce files that PreviewImport reads back without issues and with the same contents", func(t *testing.T) {
		svc := newRecipeServiceWithPicker(t, &stubFolderPicker{dir: t.TempDir()})
		orig := create(t, svc, recipe.Recipe{
			Name: "肉じゃが", Servings: 3, Tags: []string{"和食"},
			Ingredients: []recipe.RecipeIngredient{
				{Name: "じゃがいも", Amount: 3, Unit: "個"},
				{Name: "醤油", Amount: 2, Unit: "大さじ"},
				{Name: "塩", Amount: 0, Unit: "少々"},
			},
			Steps: "1. 切る\n2. 煮る", Notes: "弱火で",
		})

		got, err := svc.ExportToFolder(ctx, "title")
		if err != nil {
			t.Fatalf("ExportToFolder: %v", err)
		}
		files := exportedFiles(t, got.Dir)
		inputs := make([]recipe.ImportFile, 0, len(files))
		for name, content := range files {
			inputs = append(inputs, recipe.ImportFile{Name: name, Content: content})
		}
		candidates, err := svc.PreviewImport(ctx, inputs)
		if err != nil {
			t.Fatalf("PreviewImport: %v", err)
		}
		if len(candidates) != 1 || candidates[0].Issue != recipe.ImportIssueNone {
			t.Fatalf("candidates: %+v", candidates)
		}
		back := candidates[0].Recipe
		if back.Name != orig.Name || back.Servings != orig.Servings || back.Steps != orig.Steps || back.Notes != orig.Notes {
			t.Errorf("fields: got %+v, want %+v", back, orig)
		}
		if !reflect.DeepEqual(back.Tags, orig.Tags) || !reflect.DeepEqual(back.Ingredients, orig.Ingredients) {
			t.Errorf("tags/ingredients: got %v %v, want %v %v", back.Tags, back.Ingredients, orig.Tags, orig.Ingredients)
		}
	})
}
