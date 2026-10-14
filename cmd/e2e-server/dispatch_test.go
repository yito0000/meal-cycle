package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
	"github.com/yito0000/meal-cycle/internal/service"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

func setup(t *testing.T) http.Handler {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "e2e.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := sqlite.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return dispatchHandler(newRegistry(db, service.NewRecipeService(db)))
}

func call(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// createRecipe は後続の検証で使うレシピを 1 件作り、その ID を返す。
func createRecipe(t *testing.T, h http.Handler, name string) string {
	t.Helper()
	input := recipe.Recipe{
		Name:        name,
		Servings:    2,
		Ingredients: []recipe.RecipeIngredient{{Name: "玉ねぎ", Amount: 1, Unit: "個"}},
	}
	args, err := json.Marshal([]any{input})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	rec := call(t, h, "/api/RecipeService/Create", string(args))
	if rec.Code != http.StatusOK {
		t.Fatalf("create failed: %d %s", rec.Code, rec.Body.String())
	}
	var got recipe.Recipe
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return got.ID
}

func TestDispatchHandler(t *testing.T) {
	t.Run("should return 200 and a JSON array when a method takes no args", func(t *testing.T) {
		h := setup(t)
		rec := call(t, h, "/api/RecipeService/List", "[]")
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
		}
		var got []recipe.Recipe
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("want a JSON array, got %s", rec.Body.String())
		}
	})

	t.Run("should return 200 and a single object when a method takes one string arg", func(t *testing.T) {
		h := setup(t)
		id := createRecipe(t, h, "肉じゃが")
		rec := call(t, h, "/api/RecipeService/GetByID", `["`+id+`"]`)
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
		}
		var got recipe.Recipe
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.Name != "肉じゃが" {
			t.Errorf("want 肉じゃが, got %q", got.Name)
		}
	})

	t.Run("should hide the recipe from List when RecipeService/Delete is called", func(t *testing.T) {
		h := setup(t)
		id := createRecipe(t, h, "肉じゃが")
		rec := call(t, h, "/api/RecipeService/Delete", `["`+id+`"]`)
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
		}
		rec = call(t, h, "/api/RecipeService/List", "[]")
		var got []recipe.Recipe
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("want deleted recipe hidden, got %+v", got)
		}
	})

	t.Run("should return 200 when a method takes a string and a struct arg", func(t *testing.T) {
		h := setup(t)
		id := createRecipe(t, h, "肉じゃが")
		input := recipe.Recipe{
			Name:        "肉じゃが (改)",
			Servings:    3,
			Ingredients: []recipe.RecipeIngredient{{Name: "じゃがいも", Amount: 2, Unit: "個"}},
		}
		args, err := json.Marshal([]any{id, input})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		rec := call(t, h, "/api/RecipeService/Update", string(args))
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
		}
		var got recipe.Recipe
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.Servings != 3 {
			t.Errorf("want servings 3, got %d", got.Servings)
		}
	})

	t.Run("should preserve nested slice fields of a struct arg", func(t *testing.T) {
		h := setup(t)
		input := recipe.Recipe{
			Name:     "豚汁",
			Servings: 4,
			Tags:     []string{"汁物", "和食"},
			Ingredients: []recipe.RecipeIngredient{
				{Name: "大根", Amount: 100, Unit: "g"},
				{Name: "人参", Amount: 50, Unit: "g"},
				{Name: "味噌", Amount: 3, Unit: "大さじ"},
			},
		}
		args, err := json.Marshal([]any{input})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		rec := call(t, h, "/api/RecipeService/Create", string(args))
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
		}

		var created recipe.Recipe
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		detail := call(t, h, "/api/RecipeService/GetByID", `["`+created.ID+`"]`)
		var got recipe.Recipe
		if err := json.Unmarshal(detail.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(got.Ingredients) != 3 {
			t.Fatalf("want 3 ingredients, got %d", len(got.Ingredients))
		}
		if got.Ingredients[2].Name != "味噌" || got.Ingredients[2].Unit != "大さじ" {
			t.Errorf("nested field lost: %+v", got.Ingredients[2])
		}
	})

	t.Run("should return 200 when a method takes a slice of structs and a bool arg", func(t *testing.T) {
		h := setup(t)
		id := createRecipe(t, h, "肉じゃが")
		if rec := call(t, h, "/api/MealPlanService/AssignRecipe", `["2026-08-24","dinner","`+id+`"]`); rec.Code != http.StatusOK {
			t.Fatalf("assign failed: %d %s", rec.Code, rec.Body.String())
		}
		body := `["2026-08-24","dinner",0,[{"date":"2026-08-25","meal":"dinner"},{"date":"2026-08-26","meal":"lunch"}],"append",false]`
		rec := call(t, h, "/api/MealPlanService/SpreadEntry", body)
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
		}

		list := call(t, h, "/api/MealPlanService/ListPeriod", `["2026-08-24",1]`)
		var got []mealplan.Entry
		if err := json.Unmarshal(list.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		cells := map[string]string{}
		for _, e := range got {
			cells[e.Date+"/"+string(e.Meal)] = e.BatchID
		}
		if len(got) != 3 || cells["2026-08-25/dinner"] == "" || cells["2026-08-26/lunch"] == "" {
			t.Fatalf("want rows in the two target cells, got %+v", got)
		}
		if cells["2026-08-25/dinner"] == cells["2026-08-24/dinner"] {
			t.Errorf("shareBatch=false should mint a new batch, got the source batch")
		}
	})

	t.Run("should return 200 when a method takes no ctx and returns no error", func(t *testing.T) {
		h := setup(t)
		rec := call(t, h, "/api/HealthService/Ping", "[]")
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
		}
		var got service.Health
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !got.DBConnected {
			t.Errorf("want dbConnected true, got %+v", got)
		}
	})

	t.Run("should return 500 with kind and message when the service returns an error", func(t *testing.T) {
		h := setup(t)
		args, err := json.Marshal([]any{recipe.Recipe{Name: "", Servings: 2}})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		rec := call(t, h, "/api/RecipeService/Create", string(args))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("want 500, got %d (%s)", rec.Code, rec.Body.String())
		}
		var got map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got["kind"] == "" || got["message"] == "" {
			t.Errorf("want kind and message, got %+v", got)
		}
	})

	cases := []struct {
		name string
		path string
		body string
		want int
	}{
		{"too few args", "/api/RecipeService/GetByID", "[]", http.StatusBadRequest},
		{"too many args", "/api/RecipeService/List", `["x"]`, http.StatusBadRequest},
		{"malformed json", "/api/RecipeService/List", "{", http.StatusBadRequest},
		{"unknown service", "/api/NopeService/List", "[]", http.StatusNotFound},
		{"unknown method", "/api/RecipeService/Nope", "[]", http.StatusNotFound},
		{"malformed path", "/api/RecipeService", "[]", http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run("should return "+http.StatusText(tc.want)+" for "+tc.name, func(t *testing.T) {
			h := setup(t)
			rec := call(t, h, tc.path, tc.body)
			if rec.Code != tc.want {
				t.Errorf("want %d, got %d (%s)", tc.want, rec.Code, rec.Body.String())
			}
		})
	}
}

// main.go の application.Options.Services と registry がずれると、E2E は通るのに
// 本番で動かないサービスが生まれる。件数と名前をここで固定する。
func TestNewRegistry(t *testing.T) {
	t.Run("should register the same five services as main.go", func(t *testing.T) {
		db, err := sqlite.Open(filepath.Join(t.TempDir(), "e2e.db"))
		if err != nil {
			t.Fatalf("open db: %v", err)
		}
		defer db.Close()

		reg := newRegistry(db, service.NewRecipeService(db))
		want := []string{"HealthService", "MealPlanService", "RecipeService", "FridgeService", "ShoppingService"}
		if len(reg) != len(want) {
			t.Fatalf("want %d services, got %d", len(want), len(reg))
		}
		for _, name := range want {
			if _, ok := reg[name]; !ok {
				t.Errorf("missing service %q", name)
			}
		}
	})
}
