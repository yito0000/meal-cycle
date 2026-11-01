package main

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/domain/shopping"
	"github.com/yito0000/meal-cycle/internal/service"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

const (
	codeInternal     = "internal"
	codeInvalidInput = "input.invalid"
)

// 表に無いエラーは internal に落ちるだけで動作は止まらないので、domain のエラーを足したときの漏れは致命的ではない (design.md D6)。
var errorCodes = []struct {
	err  error
	code string
}{
	{sqlite.ErrDatabaseMissing, "database.missing"},
	{sqlite.ErrSchemaOutdated, "database.schema_outdated"},
	{sqlite.ErrSchemaTooNew, "database.schema_too_new"},

	{sqlite.ErrRecipeNotFound, "recipe.not_found"},
	{sqlite.ErrRecipeNameEmpty, "recipe.name_empty"},
	{sqlite.ErrRecipeArchived, "recipe.archived"},
	{service.ErrRecipeNameTooLong, "recipe.name_too_long"},
	{service.ErrRecipeServingsOutOfRange, "recipe.servings_out_of_range"},
	{service.ErrRecipeIngredientNameEmpty, "recipe.ingredient_name_empty"},
	{service.ErrRecipeIngredientAmountOutOfRange, "recipe.ingredient_amount_out_of_range"},
	{service.ErrRecipeStepsTooLong, "recipe.steps_too_long"},
	{service.ErrRecipeNotesTooLong, "recipe.notes_too_long"},
	{service.ErrRecipeTagTooLong, "recipe.tag_too_long"},

	{mealplan.ErrInvalidDate, "mealplan.invalid_date"},
	{mealplan.ErrInvalidMeal, "mealplan.invalid_meal"},
	{mealplan.ErrInvalidPeriodWeeks, "mealplan.invalid_period_weeks"},
	{mealplan.ErrUnknownRecipe, "mealplan.unknown_recipe"},
	{mealplan.ErrArchivedRecipe, "mealplan.archived_recipe"},
	{mealplan.ErrEntryNotFound, "mealplan.entry_not_found"},

	{fridge.ErrStockNameEmpty, "fridge.stock_name_empty"},
	{fridge.ErrStockNameTooLong, "fridge.stock_name_too_long"},
	{fridge.ErrStockAmountOutOfRange, "fridge.stock_amount_out_of_range"},
	{fridge.ErrStockUnitTooLong, "fridge.stock_unit_too_long"},
	{fridge.ErrStockInvalidExpiry, "fridge.stock_invalid_expiry"},
	{fridge.ErrStapleNameEmpty, "fridge.staple_name_empty"},
	{fridge.ErrStapleNameTooLong, "fridge.staple_name_too_long"},
	{fridge.ErrStapleDuplicate, "fridge.staple_duplicate"},
	{fridge.ErrStapleInvalidState, "fridge.staple_invalid_state"},
	{fridge.ErrStapleNotFound, "fridge.staple_not_found"},

	{shopping.ErrInvalidPeriodDays, "shopping.invalid_period_days"},
}

// toolError の Error() がそのままツールの失敗のテキストになる。エージェントが種類を判定できるよう JSON で返す。
type toolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *toolError) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}

func toToolError(err error) *toolError {
	for _, c := range errorCodes {
		if errors.Is(err, c.err) {
			return &toolError{Code: c.code, Message: err.Error()}
		}
	}
	return &toolError{Code: codeInternal, Message: err.Error()}
}

// 入力の JSON Schema で弾かれた失敗は SDK が作るのでコードが付かない。spec の「入力の形が不正なときも同じ形で返す」に揃える。
func codeInputErrors(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		res, err := next(ctx, method, req)
		if err != nil || method != "tools/call" {
			return res, err
		}
		r, ok := res.(*mcp.CallToolResult)
		if !ok || !r.IsError {
			return res, nil
		}
		var te *toolError
		if errors.As(r.GetError(), &te) {
			return res, nil
		}
		msg := ""
		if len(r.Content) > 0 {
			if tc, ok := r.Content[0].(*mcp.TextContent); ok {
				msg = tc.Text
			}
		}
		r.Content = []mcp.Content{&mcp.TextContent{Text: (&toolError{Code: codeInvalidInput, Message: msg}).Error()}}
		return r, nil
	}
}
