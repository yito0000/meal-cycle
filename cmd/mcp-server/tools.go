package main

import (
	"context"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
	"github.com/yito0000/meal-cycle/internal/domain/shopping"
	"github.com/yito0000/meal-cycle/internal/domain/suggestion"
	"github.com/yito0000/meal-cycle/internal/service"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

const serverVersion = "0.1.0"

// 削除と書き換えはエージェントに開かないので、ツールは手で選んで足す (design.md D5)。
// 増減させたら tools_test.go の一覧も直す。
func newServer(dbPath string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "meal-cycle", Version: serverVersion}, nil)
	s.AddReceivingMiddleware(codeInputErrors)
	env := toolEnv{dbPath: dbPath}

	addReadTool(s, env, "list_recipes",
		"List active (non-archived) recipes with id, name, servings and tags. Use get_recipe for ingredients and steps.",
		nil, listRecipes)
	addReadTool(s, env, "get_recipe",
		"Get one recipe by id, including ingredients, steps (Markdown), notes and cookMinutes (0 means not set).",
		nil, getRecipe)
	addReadTool(s, env, "list_meal_plan",
		"List meal plan entries from start (YYYY-MM-DD) for 1, 2 or 4 weeks. "+
			"A meal (breakfast/lunch/dinner) can hold several entries, ordered by position.",
		nil, listMealPlan)
	addReadTool(s, env, "list_stock",
		"List fridge stock items with amount, unit, expiry date (YYYY-MM-DD, empty if none) and expiry status.",
		nil, listStock)
	addReadTool(s, env, "list_staples",
		"List staple items (things kept on hand without amounts) with state enough, low or out.",
		nil, listStaples)
	addReadTool(s, env, "get_shopping_list",
		"Get the shopping list for the meal plan from start (YYYY-MM-DD) over the given number of days, "+
			"including staples whose state is low or out. "+
			"When household servings is set, amounts of one-off dishes are already scaled and scaledServings shows the household size.",
		nil, getShoppingList)
	addReadTool(s, env, "list_suggestions",
		"List recipe suggestions: byStock uses up stock close to expiry, byRecency are recipes not cooked for a while, "+
			"byTime are recipes quick to cook.",
		nil, listSuggestions)
	addReadTool(s, env, "suggest_week_plans",
		"Suggest plans that fill the empty slots of the week starting at weekStart (YYYY-MM-DD). Nothing is written.",
		nil, suggestWeekPlans)

	addWriteTool(s, env, "assign_recipe",
		"Add a recipe to a meal on a date. The entry is appended after existing entries of that meal; "+
			"existing entries are kept. Returns the created entry with its id.",
		enumSchema[assignRecipeIn]("meal", "breakfast", "lunch", "dinner"), assignRecipe)
	addWriteTool(s, env, "add_stock_items",
		"Add one or more stock items to the fridge. All items are validated first; if any item is invalid, nothing is added.",
		nil, addStockItems)
	addWriteTool(s, env, "add_staple",
		"Add a staple item by name with state enough. Fails if a staple with the same name exists.",
		nil, addStaple)
	addWriteTool(s, env, "set_staple_state",
		"Set the state of a staple item. Staples that are low or out appear in the shopping list.",
		enumSchema[setStapleStateIn]("state", "enough", "low", "out"), setStapleState)
	addWriteTool(s, env, "create_recipe",
		"Create a recipe. Ingredient amount 0 means no specific amount (e.g. a pinch of salt).",
		nil, createRecipe)
	return s
}

type toolEnv struct{ dbPath string }

type services struct {
	recipes    *service.RecipeService
	mealPlan   *service.MealPlanService
	fridge     *service.FridgeService
	shopping   *service.ShoppingService
	suggestion *service.SuggestionService
}

// 設定と DB の版をいつも DB から読むため、呼び出しごとに開いて閉じる (design.md D4)。
func (e toolEnv) run(fn func(services) error) error {
	db, err := sqlite.OpenExisting(e.dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	settings := service.NewSettingsService(db)
	return fn(services{
		recipes:    service.NewRecipeService(db, settings, noFolderPicker{}),
		mealPlan:   service.NewMealPlanService(sqlite.NewMealPlanRepo(db), sqlite.NewTemplateRepo(db), settings),
		fridge:     service.NewFridgeService(db, settings),
		shopping:   service.NewShoppingService(db, settings),
		suggestion: service.NewSuggestionService(db, settings),
	})
}

// 書き出しは公開しないので、フォルダを選ばない実装で足りる。
type noFolderPicker struct{}

func (noFolderPicker) PickFolder(string) (string, error) { return "", nil }

type toolFunc[In, Out any] func(context.Context, services, In) (Out, error)

func addReadTool[In, Out any](s *mcp.Server, env toolEnv, name, desc string, schema *jsonschema.Schema, fn toolFunc[In, Out]) {
	addTool(s, env, &mcp.Tool{
		Name: name, Description: desc,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: new(false)},
	}, schema, fn)
}

func addWriteTool[In, Out any](s *mcp.Server, env toolEnv, name, desc string, schema *jsonschema.Schema, fn toolFunc[In, Out]) {
	addTool(s, env, &mcp.Tool{
		Name: name, Description: desc,
		Annotations: &mcp.ToolAnnotations{DestructiveHint: new(false), OpenWorldHint: new(false)},
	}, schema, fn)
}

// 結果は SDK が structured content と同じ内容の JSON テキストの両方に詰める (design.md D5)。
func addTool[In, Out any](s *mcp.Server, env toolEnv, t *mcp.Tool, schema *jsonschema.Schema, fn toolFunc[In, Out]) {
	if schema != nil {
		t.InputSchema = schema
	}
	mcp.AddTool(s, t, func(ctx context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		var out Out
		err := env.run(func(svc services) error {
			var err error
			out, err = fn(ctx, svc, in)
			return err
		})
		if err != nil {
			var zero Out
			return nil, zero, toToolError(err)
		}
		return nil, out, nil
	})
}

// jsonschema-go の struct タグでは enum を書けないので、推論した schema に後から足す。
func enumSchema[In any](field string, values ...string) *jsonschema.Schema {
	sc, err := jsonschema.For[In](nil)
	if err != nil {
		panic(fmt.Sprintf("infer schema: %v", err))
	}
	prop, ok := sc.Properties[field]
	if !ok {
		panic(fmt.Sprintf("schema has no property %q", field))
	}
	for _, v := range values {
		prop.Enum = append(prop.Enum, v)
	}
	return sc
}

// --- 参照 ---

type noInput struct{}

type recipeSummary struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Servings int      `json:"servings"`
	Tags     []string `json:"tags"`
}

type listRecipesOut struct {
	Recipes []recipeSummary `json:"recipes"`
}

func listRecipes(ctx context.Context, svc services, _ noInput) (listRecipesOut, error) {
	rs, err := svc.recipes.List(ctx)
	if err != nil {
		return listRecipesOut{}, err
	}
	out := listRecipesOut{Recipes: make([]recipeSummary, 0, len(rs))}
	for _, r := range rs {
		out.Recipes = append(out.Recipes, recipeSummary{ID: r.ID, Name: r.Name, Servings: r.Servings, Tags: r.Tags})
	}
	return out, nil
}

type getRecipeIn struct {
	ID string `json:"id" jsonschema:"recipe id from list_recipes"`
}

func getRecipe(ctx context.Context, svc services, in getRecipeIn) (recipe.Recipe, error) {
	return svc.recipes.GetByID(ctx, in.ID)
}

type listMealPlanIn struct {
	Start string `json:"start" jsonschema:"first date, YYYY-MM-DD"`
	Weeks int    `json:"weeks" jsonschema:"number of weeks: 1, 2 or 4"`
}

type listMealPlanOut struct {
	Entries []mealplan.Entry `json:"entries"`
}

func listMealPlan(ctx context.Context, svc services, in listMealPlanIn) (listMealPlanOut, error) {
	es, err := svc.mealPlan.ListPeriod(ctx, in.Start, in.Weeks)
	if err != nil {
		return listMealPlanOut{}, err
	}
	return listMealPlanOut{Entries: nonNil(es)}, nil
}

type stockItemsOut struct {
	Items []fridge.StockItem `json:"items"`
}

func listStock(ctx context.Context, svc services, _ noInput) (stockItemsOut, error) {
	items, err := svc.fridge.List(ctx)
	if err != nil {
		return stockItemsOut{}, err
	}
	return stockItemsOut{Items: nonNil(items)}, nil
}

type listStaplesOut struct {
	Staples []fridge.Staple `json:"staples"`
}

func listStaples(ctx context.Context, svc services, _ noInput) (listStaplesOut, error) {
	st, err := svc.fridge.ListStaples(ctx)
	if err != nil {
		return listStaplesOut{}, err
	}
	return listStaplesOut{Staples: nonNil(st)}, nil
}

type getShoppingListIn struct {
	Start string `json:"start" jsonschema:"first date, YYYY-MM-DD"`
	Days  int    `json:"days" jsonschema:"number of days from start"`
}

func getShoppingList(ctx context.Context, svc services, in getShoppingListIn) (shopping.List, error) {
	return svc.shopping.List(ctx, in.Start, in.Days)
}

func listSuggestions(ctx context.Context, svc services, _ noInput) (suggestion.Result, error) {
	return svc.suggestion.List(ctx)
}

type suggestWeekPlansIn struct {
	WeekStart string `json:"weekStart" jsonschema:"first date of the week, YYYY-MM-DD"`
}

func suggestWeekPlans(ctx context.Context, svc services, in suggestWeekPlansIn) (suggestion.WeekPlanResult, error) {
	return svc.suggestion.WeekPlans(ctx, in.WeekStart)
}

// --- 追加と状態の変更 ---

type assignRecipeIn struct {
	Date     string `json:"date" jsonschema:"YYYY-MM-DD"`
	Meal     string `json:"meal" jsonschema:"breakfast, lunch or dinner"`
	RecipeID string `json:"recipeId" jsonschema:"recipe id from list_recipes"`
}

func assignRecipe(ctx context.Context, svc services, in assignRecipeIn) (mealplan.Entry, error) {
	return svc.mealPlan.AssignRecipe(ctx, in.Date, in.Meal, in.RecipeID)
}

type stockItemIn struct {
	Name      string  `json:"name" jsonschema:"ingredient name"`
	Amount    float64 `json:"amount" jsonschema:"amount, greater than 0"`
	Unit      string  `json:"unit,omitempty" jsonschema:"unit such as g, ml or 個"`
	ExpiresAt string  `json:"expiresAt,omitempty" jsonschema:"expiry date YYYY-MM-DD; omit if none"`
}

type addStockItemsIn struct {
	Items []stockItemIn `json:"items" jsonschema:"stock items to add"`
}

func addStockItems(ctx context.Context, svc services, in addStockItemsIn) (stockItemsOut, error) {
	inputs := make([]fridge.StockItem, 0, len(in.Items))
	for _, it := range in.Items {
		inputs = append(inputs, fridge.StockItem{Name: it.Name, Amount: it.Amount, Unit: it.Unit, ExpiresAt: it.ExpiresAt})
	}
	created, err := svc.fridge.AddMany(ctx, inputs)
	if err != nil {
		return stockItemsOut{}, err
	}
	return stockItemsOut{Items: nonNil(created)}, nil
}

type addStapleIn struct {
	Name string `json:"name" jsonschema:"staple name"`
}

func addStaple(ctx context.Context, svc services, in addStapleIn) (fridge.Staple, error) {
	return svc.fridge.AddStaple(ctx, in.Name)
}

type setStapleStateIn struct {
	ID    string `json:"id" jsonschema:"staple id from list_staples"`
	State string `json:"state" jsonschema:"enough, low or out"`
}

func setStapleState(ctx context.Context, svc services, in setStapleStateIn) (fridge.Staple, error) {
	return svc.fridge.SetStapleState(ctx, in.ID, fridge.StapleState(in.State))
}

type ingredientIn struct {
	Name   string  `json:"name" jsonschema:"ingredient name"`
	Amount float64 `json:"amount,omitempty" jsonschema:"amount; 0 or omitted means no specific amount"`
	Unit   string  `json:"unit,omitempty" jsonschema:"unit such as g, ml or 大さじ"`
}

type createRecipeIn struct {
	Name        string         `json:"name" jsonschema:"recipe name"`
	Servings    int            `json:"servings" jsonschema:"number of servings, 1 to 99"`
	CookMinutes int            `json:"cookMinutes,omitempty" jsonschema:"cooking time in minutes, 1 to 999; 0 or omitted means not set"`
	Tags        []string       `json:"tags,omitempty"`
	Ingredients []ingredientIn `json:"ingredients,omitempty"`
	Steps       string         `json:"steps,omitempty" jsonschema:"steps in Markdown"`
	Notes       string         `json:"notes,omitempty"`
}

func createRecipe(ctx context.Context, svc services, in createRecipeIn) (recipe.Recipe, error) {
	ings := make([]recipe.RecipeIngredient, 0, len(in.Ingredients))
	for _, ing := range in.Ingredients {
		ings = append(ings, recipe.RecipeIngredient{Name: ing.Name, Amount: ing.Amount, Unit: ing.Unit})
	}
	return svc.recipes.Create(ctx, recipe.Recipe{
		Name: in.Name, Servings: in.Servings, CookMinutes: in.CookMinutes, Tags: in.Tags, Ingredients: ings,
		Steps: in.Steps, Notes: in.Notes,
	})
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
