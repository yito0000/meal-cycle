package mealplan

import (
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrTemplateNameEmpty     = errors.New("mealplan: template name empty")
	ErrTemplateNameTooLong   = errors.New("mealplan: template name too long")
	ErrTemplateNameDuplicate = errors.New("mealplan: template name duplicate")
	ErrTemplateEmptyWeek     = errors.New("mealplan: template empty week")
	ErrTemplateNotFound      = errors.New("mealplan: template not found")
	ErrTemplateAllArchived   = errors.New("mealplan: template all archived")
)

// テンプレートの名前の上限。一覧のダイアログで 1 行に収まる長さで、利用者が変える値ではない。
const TemplateNameMaxRunes = 50

// テンプレートは 1 週間の組み方なので、ずれは 0〜6 日。
const templateDays = 7

// Template は名前を付けて保存した 1 週間の組み方 (add-mealplan-template design.md D1)。
type Template struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	CreatedAt string         `json:"createdAt"`
	Items     []TemplateItem `json:"items"`
}

// TemplateItem はテンプレートの 1 品。DayOffset は週の最初の日からの日数、GroupNo は同じ作り置きの組の番号。
// RecipeName と Archived は一覧の表示用で、保存はしない。
type TemplateItem struct {
	DayOffset  int    `json:"dayOffset"`
	Meal       Meal   `json:"meal"`
	Position   int    `json:"position"`
	RecipeID   string `json:"recipeId"`
	GroupNo    int    `json:"groupNo"`
	RecipeName string `json:"recipeName"`
	Archived   bool   `json:"archived"`
}

// TemplateApplyResult は ApplyTemplate の返り値。Result を UndoSpread に渡すと適用を取り消せる。
type TemplateApplyResult struct {
	Result          SpreadResult `json:"result"`
	SkippedArchived int          `json:"skippedArchived"`
}

// ValidateTemplateName は前後の空白を除いた名前を返す。
func ValidateTemplateName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", ErrTemplateNameEmpty
	}
	if utf8.RuneCountInString(trimmed) > TemplateNameMaxRunes {
		return "", ErrTemplateNameTooLong
	}
	return trimmed, nil
}

// TemplateItemsFromWeek は weekStart から 7 日間の行をテンプレートの行にする。
// batch_id は週の中で最初に出てくる順に 0, 1, 2… の組の番号に振り直すので、週の外に続く作り置きは週の中の行だけが同じ組になる。
func TemplateItemsFromWeek(weekStart string, entries []Entry) []TemplateItem {
	start, err := time.Parse(dateLayout, weekStart)
	if err != nil {
		return []TemplateItem{}
	}
	inWeek := make([]TemplateItem, 0, len(entries))
	batches := make([]string, 0, len(entries))
	for _, e := range entries {
		d, err := time.Parse(dateLayout, e.Date)
		if err != nil {
			continue
		}
		offset := int(d.Sub(start).Hours() / 24)
		if offset < 0 || offset >= templateDays {
			continue
		}
		inWeek = append(inWeek, TemplateItem{
			DayOffset: offset, Meal: e.Meal, Position: e.Position, RecipeID: e.RecipeID, RecipeName: e.RecipeName,
		})
		batches = append(batches, e.BatchID)
	}
	order := make([]int, len(inWeek))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return lessItem(inWeek[order[a]], inWeek[order[b]]) })

	groups := map[string]int{}
	out := make([]TemplateItem, 0, len(inWeek))
	for _, i := range order {
		it := inWeek[i]
		g, ok := groups[batches[i]]
		if !ok {
			g = len(groups)
			groups[batches[i]] = g
		}
		it.GroupNo = g
		out = append(out, it)
	}
	return out
}

// PlanItemsFromTemplate はテンプレートの行を weekStart の週の PlanItem にする。
// 削除済みのレシピの行は除き、除いた数を返す。同じセルの品がテンプレートの順で末尾に積まれるよう、(日, 時間帯, 順) に並べる。
func PlanItemsFromTemplate(weekStart string, items []TemplateItem, archived map[string]bool) ([]PlanItem, int) {
	start, err := time.Parse(dateLayout, weekStart)
	if err != nil {
		return []PlanItem{}, 0
	}
	sorted := append([]TemplateItem(nil), items...)
	sort.SliceStable(sorted, func(a, b int) bool { return lessItem(sorted[a], sorted[b]) })
	out := make([]PlanItem, 0, len(sorted))
	skipped := 0
	for _, it := range sorted {
		if archived[it.RecipeID] {
			skipped++
			continue
		}
		out = append(out, PlanItem{
			Date:     start.AddDate(0, 0, it.DayOffset).Format(dateLayout),
			Meal:     it.Meal,
			RecipeID: it.RecipeID,
			Group:    it.GroupNo,
		})
	}
	return out, skipped
}

var mealRank = map[Meal]int{MealBreakfast: 0, MealLunch: 1, MealDinner: 2}

func lessItem(a, b TemplateItem) bool {
	if a.DayOffset != b.DayOffset {
		return a.DayOffset < b.DayOffset
	}
	if mealRank[a.Meal] != mealRank[b.Meal] {
		return mealRank[a.Meal] < mealRank[b.Meal]
	}
	return a.Position < b.Position
}
