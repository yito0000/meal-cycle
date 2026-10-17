package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yito0000/meal-cycle/internal/config"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

func newRecipeServiceForTest(t *testing.T) *RecipeService {
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
	return NewRecipeService(db, staticConfig(config.Defaults()))
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
}
