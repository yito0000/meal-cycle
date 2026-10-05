package shopping

import (
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
)

// 表示 (formatAmount) と同じ小数第 2 位で丸めてから比較し、0.1+0.2 と 0.3 の誤差で partial に倒れるのを防ぐ。
const amountScale = 100

var mealOrder = map[string]int{
	string(mealplan.MealBreakfast): 0,
	string(mealplan.MealLunch):     1,
	string(mealplan.MealDinner):    2,
}

func roundAmount(v float64) float64 {
	return math.Round(v*amountScale) / amountScale
}

type ingredientKey struct {
	name string
	unit string
}

// Aggregate は計上する作り置きの材料を「表記をそろえた食材名 + 単位」で合算し、在庫の状態を付けて並べる。
// 量 0 と量る単位の材料は量を持たない材料として、食材名 + 空の単位に量 0 で合算する (add-measuring-units design.md D3)。
func Aggregate(batches []PlannedBatch, stocks []StockAmount, measuringUnits []string) []Item {
	stockByKey := map[ingredientKey]float64{}
	stockByName := map[string]float64{}
	for _, s := range stocks {
		name := normalizeKey(s.Name)
		stockByKey[ingredientKey{name, normalizeKey(s.Unit)}] += s.Amount
		stockByName[name] += s.Amount
	}
	measuring := make([]string, 0, len(measuringUnits))
	for _, u := range measuringUnits {
		measuring = append(measuring, normalizeKey(u))
	}

	// 用途欄のレシピ名を献立一覧と同じ順に並べるため、先に献立の順へ並べ替えてから走査する。
	ordered := sortedBatches(batches)
	index := map[ingredientKey]int{}
	// 表示は最初に出てくる材料の表記を使うので、在庫を引くためのキーは行と別に持つ (add-ingredient-name-matching design.md D3)。
	keys := make([]ingredientKey, 0)
	items := make([]Item, 0)
	for _, b := range ordered {
		for _, ing := range b.Ingredients {
			k := ingredientKey{normalizeKey(ing.Name), normalizeKey(ing.Unit)}
			unit := strings.TrimSpace(ing.Unit)
			amount := ing.Amount
			if amount == 0 || slices.Contains(measuring, k.unit) {
				k.unit = ""
				unit = ""
				amount = 0
			}
			i, ok := index[k]
			if !ok {
				i = len(items)
				index[k] = i
				keys = append(keys, k)
				items = append(items, Item{Name: strings.TrimSpace(ing.Name), Unit: unit, Recipes: []string{}, firstDate: b.FirstDate})
			}
			items[i].Amount += amount
			if !containsString(items[i].Recipes, b.RecipeName) {
				items[i].Recipes = append(items[i].Recipes, b.RecipeName)
			}
		}
	}

	for i := range items {
		items[i].Amount = roundAmount(items[i].Amount)
		stock := roundAmount(stockByKey[keys[i]])
		// 量を持たない行は単位を問わず同名の在庫を数える。単位の無い量のある行に合算された行は量のある行として扱う (design.md D4)。
		if items[i].Amount == 0 {
			stock = roundAmount(stockByName[keys[i].name])
		}
		items[i].StockStatus = stockStatus(items[i].Amount, stock)
	}

	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if (a.StockStatus == StockCovered) != (b.StockStatus == StockCovered) {
			return b.StockStatus == StockCovered
		}
		if a.firstDate != b.firstDate {
			return a.firstDate < b.firstDate
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Unit < b.Unit
	})
	return items
}

// 量 0 の材料 (required == 0) も同じ式で none / covered に振り分けられる (design.md D5)。
func stockStatus(required, stock float64) StockStatus {
	switch {
	case stock <= 0:
		return StockNone
	case stock < required:
		return StockPartial
	default:
		return StockCovered
	}
}

// Dishes は S07「この期間の献立」欄の行を、期間内で最初に出てくる行の (日付, 時間帯, セル内の並び) 順に返す。
func Dishes(batches []PlannedBatch) []Dish {
	ordered := sortedBatches(batches)
	out := make([]Dish, 0, len(ordered))
	for _, b := range ordered {
		out = append(out, Dish{
			BatchID:             b.BatchID,
			RecipeID:            b.RecipeID,
			RecipeName:          b.RecipeName,
			Dates:               nonNil(b.Dates),
			Meals:               nonNil(b.Meals),
			BatchMemberCount:    b.BatchMemberCount,
			StartedBeforePeriod: b.StartedBeforePeriod,
		})
	}
	return out
}

func sortedBatches(batches []PlannedBatch) []PlannedBatch {
	out := append([]PlannedBatch(nil), batches...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.FirstDate != b.FirstDate {
			return a.FirstDate < b.FirstDate
		}
		if ma, mb := mealRank(a.FirstMeal), mealRank(b.FirstMeal); ma != mb {
			return ma < mb
		}
		if a.FirstPosition != b.FirstPosition {
			return a.FirstPosition < b.FirstPosition
		}
		return a.RecipeName < b.RecipeName
	})
	return out
}

// SortMeals は時間帯を朝 / 昼 / 夕の順に並べ替える (未知の値は末尾)。
func SortMeals(meals []string) {
	sort.SliceStable(meals, func(i, j int) bool {
		return mealRank(meals[i]) < mealRank(meals[j])
	})
}

func mealRank(m string) int {
	if o, ok := mealOrder[m]; ok {
		return o
	}
	return len(mealOrder)
}

func containsString(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func nonNil(xs []string) []string {
	out := make([]string, len(xs))
	copy(out, xs)
	return out
}
