package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
	"github.com/yito0000/meal-cycle/internal/domain/shopping"
	"github.com/yito0000/meal-cycle/internal/service"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

type fixture struct {
	cs   *mcp.ClientSession
	db   *sql.DB
	path string
}

// setup は最新の版の DB を作り、in-memory の transport で MCP サーバーとクライアントを繋ぐ。
func setup(t *testing.T) fixture {
	t.Helper()
	path := filepath.Join(t.TempDir(), "meal-cycle.db")
	db, err := sqlite.OpenAndMigrate(path)
	if err != nil {
		t.Fatalf("OpenAndMigrate: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return fixture{cs: connect(t, path), db: db, path: path}
}

func connect(t *testing.T, path string) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := newServer(path).Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() {
		_ = cs.Close()
		_ = ss.Wait()
	})
	return cs
}

func call(t *testing.T, cs *mcp.ClientSession, name string, args any) *mcp.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	return res
}

func text(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("no content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content[0] is %T, want *TextContent", res.Content[0])
	}
	return tc.Text
}

// callOK は成功を確かめ、JSON テキストを out に読む。
func callOK[T any](t *testing.T, cs *mcp.ClientSession, name string, args any) T {
	t.Helper()
	res := call(t, cs, name, args)
	if res.IsError {
		t.Fatalf("%s failed: %s", name, text(t, res))
	}
	var out T
	if err := json.Unmarshal([]byte(text(t, res)), &out); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return out
}

// callErr は失敗を確かめ、返ったコードを返す。
func callErr(t *testing.T, cs *mcp.ClientSession, name string, args any) string {
	t.Helper()
	res := call(t, cs, name, args)
	if !res.IsError {
		t.Fatalf("%s: want error, got %s", name, text(t, res))
	}
	var te toolError
	if err := json.Unmarshal([]byte(text(t, res)), &te); err != nil {
		t.Fatalf("decode error of %s: %v (%s)", name, err, text(t, res))
	}
	if te.Message == "" {
		t.Errorf("%s: error message is empty", name)
	}
	return te.Code
}

func seedRecipe(t *testing.T, cs *mcp.ClientSession, name string) string {
	t.Helper()
	return callOK[recipe.Recipe](t, cs, "create_recipe", map[string]any{"name": name, "servings": 2}).ID
}

func listMeal(t *testing.T, cs *mcp.ClientSession, start string) []mealplan.Entry {
	t.Helper()
	return callOK[listMealPlanOut](t, cs, "list_meal_plan", map[string]any{"start": start, "weeks": 1}).Entries
}

func schemaVersion(t *testing.T, path string) int64 {
	t.Helper()
	db, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var v int64
	if err := db.QueryRow(`SELECT MAX(version_id) FROM goose_db_version`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

var wantTools = []string{
	"add_staple", "add_stock_items", "assign_recipe", "create_recipe",
	"get_recipe", "get_shopping_list", "list_meal_plan", "list_recipes",
	"list_staples", "list_stock", "list_suggestions", "set_staple_state", "suggest_week_plans",
}

func TestMCPServer_ListTools(t *testing.T) {
	listNames := func(t *testing.T) []string {
		f := setup(t)
		res, err := f.cs.ListTools(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, tool := range res.Tools {
			if tool.Description == "" {
				t.Errorf("tool %s has no description", tool.Name)
			}
			names = append(names, tool.Name)
		}
		slices.Sort(names)
		return names
	}

	t.Run("should return exactly the 13 tools in design.md D5 when the client lists tools", func(t *testing.T) {
		if got := listNames(t); !slices.Equal(got, wantTools) {
			t.Errorf("tools = %v, want %v", got, wantTools)
		}
	})

	t.Run("should not expose any delete or update tool when the client lists tools", func(t *testing.T) {
		for _, name := range listNames(t) {
			for _, verb := range []string{"delete", "remove", "update", "edit", "clear", "replace", "move", "consume", "restore", "import", "export", "save"} {
				if strings.Contains(name, verb) {
					t.Errorf("tool %s looks like a %s operation", name, verb)
				}
			}
		}
	})
}

func TestMCPServer_DatabaseCheck(t *testing.T) {
	t.Run("should fail with database.missing and not create the file when the database does not exist", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "meal-cycle.db")
		cs := connect(t, path)

		if code := callErr(t, cs, "list_recipes", map[string]any{}); code != "database.missing" {
			t.Errorf("code = %q, want database.missing", code)
		}
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("file should not be created, stat err = %v", err)
		}
	})

	t.Run("should fail with database.schema_outdated and keep the version when the schema is old", func(t *testing.T) {
		f := setup(t)
		if _, err := f.db.Exec(`DELETE FROM goose_db_version WHERE version_id > 3`); err != nil {
			t.Fatal(err)
		}

		if code := callErr(t, f.cs, "list_recipes", map[string]any{}); code != "database.schema_outdated" {
			t.Errorf("code = %q, want database.schema_outdated", code)
		}
		if v := schemaVersion(t, f.path); v != 3 {
			t.Errorf("version = %d, want 3", v)
		}
		if _, err := os.Stat(sqlite.BackupDir(f.path)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("backups dir should not be created, stat err = %v", err)
		}
	})

	t.Run("should fail with database.schema_too_new when the schema is newer than the server", func(t *testing.T) {
		f := setup(t)
		next := schemaVersion(t, f.path) + 1
		if _, err := f.db.Exec(`INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, 1)`, next); err != nil {
			t.Fatal(err)
		}

		if code := callErr(t, f.cs, "list_recipes", map[string]any{}); code != "database.schema_too_new" {
			t.Errorf("code = %q, want database.schema_too_new", code)
		}
	})
}

func TestMCPServer_Read(t *testing.T) {
	t.Run("should return the assigned entry when listing a week of meal plan", func(t *testing.T) {
		f := setup(t)
		id := seedRecipe(t, f.cs, "肉じゃが")
		callOK[mealplan.Entry](t, f.cs, "assign_recipe", map[string]any{"date": "2026-11-02", "meal": "dinner", "recipeId": id})

		got := listMeal(t, f.cs, "2026-11-02")
		if len(got) != 1 {
			t.Fatalf("entries = %d, want 1", len(got))
		}
		if got[0].Date != "2026-11-02" || got[0].Meal != mealplan.MealDinner || got[0].RecipeName != "肉じゃが" {
			t.Errorf("entry = %+v", got[0])
		}
	})

	t.Run("should fail with mealplan.invalid_period_weeks when weeks is 3", func(t *testing.T) {
		f := setup(t)
		code := callErr(t, f.cs, "list_meal_plan", map[string]any{"start": "2026-11-02", "weeks": 3})
		if code != "mealplan.invalid_period_weeks" {
			t.Errorf("code = %q, want mealplan.invalid_period_weeks", code)
		}
	})

	t.Run("should fail with recipe.not_found when getting an unknown recipe", func(t *testing.T) {
		f := setup(t)
		if code := callErr(t, f.cs, "get_recipe", map[string]any{"id": "no-such-id"}); code != "recipe.not_found" {
			t.Errorf("code = %q, want recipe.not_found", code)
		}
	})

	t.Run("should return recipes without ingredients and steps when listing recipes", func(t *testing.T) {
		f := setup(t)
		callOK[recipe.Recipe](t, f.cs, "create_recipe", map[string]any{
			"name": "豚汁", "servings": 2, "tags": []string{"汁物"},
			"ingredients": []map[string]any{{"name": "豚こま", "amount": 100, "unit": "g"}},
			"steps":       "煮る",
		})

		res := call(t, f.cs, "list_recipes", map[string]any{})
		var raw struct {
			Recipes []map[string]any `json:"recipes"`
		}
		if err := json.Unmarshal([]byte(text(t, res)), &raw); err != nil {
			t.Fatal(err)
		}
		if len(raw.Recipes) != 1 {
			t.Fatalf("recipes = %d, want 1", len(raw.Recipes))
		}
		r := raw.Recipes[0]
		if r["name"] != "豚汁" || r["servings"] != float64(2) {
			t.Errorf("recipe = %v", r)
		}
		for _, k := range []string{"ingredients", "steps", "notes"} {
			if _, ok := r[k]; ok {
				t.Errorf("recipe has %q, want summary only", k)
			}
		}
	})
}

func TestMCPServer_AssignRecipe(t *testing.T) {
	t.Run("should append the entry after existing ones when the meal already has an entry", func(t *testing.T) {
		f := setup(t)
		first := seedRecipe(t, f.cs, "肉じゃが")
		second := seedRecipe(t, f.cs, "味噌汁")
		callOK[mealplan.Entry](t, f.cs, "assign_recipe", map[string]any{"date": "2026-11-02", "meal": "dinner", "recipeId": first})

		added := callOK[mealplan.Entry](t, f.cs, "assign_recipe", map[string]any{"date": "2026-11-02", "meal": "dinner", "recipeId": second})
		if added.ID == "" {
			t.Error("want entry id")
		}

		got := listMeal(t, f.cs, "2026-11-02")
		var names []string
		for _, e := range got {
			names = append(names, e.RecipeName)
		}
		if !slices.Equal(names, []string{"肉じゃが", "味噌汁"}) {
			t.Errorf("dinner = %v, want [肉じゃが 味噌汁]", names)
		}
	})

	t.Run("should fail and keep the meal plan when the recipe does not exist", func(t *testing.T) {
		f := setup(t)
		code := callErr(t, f.cs, "assign_recipe", map[string]any{"date": "2026-11-02", "meal": "dinner", "recipeId": "no-such-id"})
		if code != "mealplan.unknown_recipe" {
			t.Errorf("code = %q, want mealplan.unknown_recipe", code)
		}
		if got := listMeal(t, f.cs, "2026-11-02"); len(got) != 0 {
			t.Errorf("entries = %d, want 0", len(got))
		}
	})

	t.Run(`should fail with mealplan.invalid_date and keep the meal plan when the date is "11/2"`, func(t *testing.T) {
		f := setup(t)
		id := seedRecipe(t, f.cs, "肉じゃが")
		code := callErr(t, f.cs, "assign_recipe", map[string]any{"date": "11/2", "meal": "dinner", "recipeId": id})
		if code != "mealplan.invalid_date" {
			t.Errorf("code = %q, want mealplan.invalid_date", code)
		}
		if got := listMeal(t, f.cs, "2026-11-02"); len(got) != 0 {
			t.Errorf("entries = %d, want 0", len(got))
		}
	})

	t.Run("should fail with input.invalid when the meal is not one of the enum values", func(t *testing.T) {
		f := setup(t)
		id := seedRecipe(t, f.cs, "肉じゃが")
		code := callErr(t, f.cs, "assign_recipe", map[string]any{"date": "2026-11-02", "meal": "brunch", "recipeId": id})
		if code != codeInvalidInput {
			t.Errorf("code = %q, want %s", code, codeInvalidInput)
		}
	})
}

func TestMCPServer_AddStockItems(t *testing.T) {
	t.Run("should add all items when every item is valid", func(t *testing.T) {
		f := setup(t)
		out := callOK[stockItemsOut](t, f.cs, "add_stock_items", map[string]any{"items": []map[string]any{
			{"name": "玉ねぎ", "amount": 3, "unit": "個"},
			{"name": "豚こま", "amount": 300, "unit": "g", "expiresAt": "2026-11-05"},
		}})
		if len(out.Items) != 2 || out.Items[0].ID == "" {
			t.Fatalf("items = %+v", out.Items)
		}
		if got := callOK[stockItemsOut](t, f.cs, "list_stock", map[string]any{}); len(got.Items) != 2 {
			t.Errorf("stock = %d, want 2", len(got.Items))
		}
	})

	t.Run("should fail with fridge.stock_name_empty and add nothing when one item has an empty name", func(t *testing.T) {
		f := setup(t)
		code := callErr(t, f.cs, "add_stock_items", map[string]any{"items": []map[string]any{
			{"name": "玉ねぎ", "amount": 3, "unit": "個"},
			{"name": "", "amount": 1},
		}})
		if code != "fridge.stock_name_empty" {
			t.Errorf("code = %q, want fridge.stock_name_empty", code)
		}
		if got := callOK[stockItemsOut](t, f.cs, "list_stock", map[string]any{}); len(got.Items) != 0 {
			t.Errorf("stock = %d, want 0", len(got.Items))
		}
	})
}

func TestMCPServer_Staples(t *testing.T) {
	t.Run("should list the staple in the shopping list when its state is set to out", func(t *testing.T) {
		f := setup(t)
		milk := callOK[fridge.Staple](t, f.cs, "add_staple", map[string]any{"name": "牛乳"})

		got := callOK[fridge.Staple](t, f.cs, "set_staple_state", map[string]any{"id": milk.ID, "state": "out"})
		if got.State != fridge.StapleOut {
			t.Errorf("state = %q, want out", got.State)
		}

		list := callOK[shopping.List](t, f.cs, "get_shopping_list", map[string]any{"start": "2026-11-02", "days": 7})
		if !slices.ContainsFunc(list.Staples, func(s shopping.StapleItem) bool { return s.Name == "牛乳" }) {
			t.Errorf("staples = %+v, want 牛乳", list.Staples)
		}
	})

	t.Run("should fail with fridge.staple_duplicate when adding a staple with an existing name", func(t *testing.T) {
		f := setup(t)
		callOK[fridge.Staple](t, f.cs, "add_staple", map[string]any{"name": "牛乳"})
		if code := callErr(t, f.cs, "add_staple", map[string]any{"name": "牛乳"}); code != "fridge.staple_duplicate" {
			t.Errorf("code = %q, want fridge.staple_duplicate", code)
		}
	})
}

func TestMCPServer_CreateRecipe(t *testing.T) {
	t.Run("should return the created recipe and include it in list_recipes when the input is valid", func(t *testing.T) {
		f := setup(t)
		created := callOK[recipe.Recipe](t, f.cs, "create_recipe", map[string]any{
			"name": "豚汁", "servings": 2,
			"ingredients": []map[string]any{{"name": "豚こま", "amount": 100, "unit": "g"}},
		})
		if created.ID == "" || created.Name != "豚汁" || len(created.Ingredients) != 1 {
			t.Fatalf("created = %+v", created)
		}

		list := callOK[listRecipesOut](t, f.cs, "list_recipes", map[string]any{})
		if !slices.ContainsFunc(list.Recipes, func(r recipeSummary) bool { return r.ID == created.ID && r.Name == "豚汁" }) {
			t.Errorf("recipes = %+v, want 豚汁", list.Recipes)
		}
	})
}

func TestMCPServer_Settings(t *testing.T) {
	t.Run("should judge expiry status with the saved red days when settings are saved after the server started", func(t *testing.T) {
		f := setup(t)
		expires := time.Now().AddDate(0, 0, 5).Format(fridge.DateLayout)
		callOK[stockItemsOut](t, f.cs, "add_stock_items", map[string]any{"items": []map[string]any{
			{"name": "豚こま", "amount": 300, "unit": "g", "expiresAt": expires},
		}})
		// 既定 (🔴 3 日 / 🟡 7 日) では残り 5 日は soon。
		if got := callOK[stockItemsOut](t, f.cs, "list_stock", map[string]any{}); got.Items[0].ExpiryStatus != fridge.ExpirySoon {
			t.Fatalf("status before = %q, want soon", got.Items[0].ExpiryStatus)
		}

		ctx := context.Background()
		gui := service.NewSettingsService(f.db)
		values, err := gui.Get(ctx)
		if err != nil {
			t.Fatal(err)
		}
		values.ExpiryAlertRedDays = 5
		if _, err := gui.Save(ctx, values); err != nil {
			t.Fatal(err)
		}

		if got := callOK[stockItemsOut](t, f.cs, "list_stock", map[string]any{}); got.Items[0].ExpiryStatus != fridge.ExpiryUrgent {
			t.Errorf("status after = %q, want urgent", got.Items[0].ExpiryStatus)
		}
	})
}
