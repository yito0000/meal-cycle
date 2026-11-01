package service

import (
	"context"
	"database/sql"
	"slices"
	"sort"
	"time"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
	"github.com/yito0000/meal-cycle/internal/domain/suggestion"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// SuggestionService は S01 のレシピ選択ダイアログに出す献立候補をフロントに公開する。
// 在庫、レシピ、献立の履歴を読む依存をこのサービスに閉じる (add-mealplan-suggest-by-stock design.md D1)。
type SuggestionService struct {
	recipes *sqlite.RecipeRepo
	stock   *sqlite.StockRepo
	history *sqlite.SuggestionRepo
	meals   *sqlite.MealPlanRepo
	cfg     configSource
	now     func() time.Time
}

func NewSuggestionService(db *sql.DB, cfg configSource) *SuggestionService {
	return &SuggestionService{
		recipes: sqlite.NewRecipeRepo(db),
		stock:   sqlite.NewStockRepo(db),
		history: sqlite.NewSuggestionRepo(db),
		meals:   sqlite.NewMealPlanRepo(db),
		cfg:     cfg,
		now:     time.Now,
	}
}

// List は全軸の候補を 1 回で返す。期限は今日を起点に数え、セルの日付には合わせない。
func (s *SuggestionService) List(ctx context.Context) (suggestion.Result, error) {
	recipes, err := s.recipes.ListDetailed(ctx)
	if err != nil {
		return suggestion.Result{}, err
	}
	stocks, err := s.stock.List(ctx)
	if err != nil {
		return suggestion.Result{}, err
	}
	today := s.now()
	// 在庫の期限の判定と同じ「今日」(ローカル日付) で予定の有無を見る。
	history, err := s.history.ListRecipeHistory(ctx, today.Format(fridge.DateLayout))
	if err != nil {
		return suggestion.Result{}, err
	}
	cfg := s.cfg.current()
	return suggestion.Result{
		ByStock:   suggestion.ByStock(recipes, stocks, today, cfg.ExpiryAlertRedDays, cfg.ExpiryAlertYellowDays, cfg.SuggestionLimit),
		ByRecency: suggestion.ByRecency(recipes, history, today, cfg.RecencyThresholdDays, cfg.SuggestionLimit),
	}, nil
}

// weekDays は週の候補の対象の日数。表示中の期間の最初の 1 週間だけを埋める (add-mealplan-suggest-week-preset proposal Non-goals)。
const weekDays = 7

// WeekPlans は weekStart から 7 日間の空いたコマを、少数のレシピの作り置きで埋めた案を返す。
func (s *SuggestionService) WeekPlans(ctx context.Context, weekStart string) (suggestion.WeekPlanResult, error) {
	if err := mealplan.ValidateDate(weekStart); err != nil {
		return suggestion.WeekPlanResult{}, err
	}
	cfg := s.cfg.current()
	slots, err := s.emptySlots(ctx, weekStart, cfg.WeeklyFillDays, cfg.WeeklyFillMeal)
	if err != nil {
		return suggestion.WeekPlanResult{}, err
	}
	result := suggestion.WeekPlanResult{Slots: slots, Plans: []suggestion.WeekPlan{}}
	if len(slots) == 0 {
		return result, nil
	}

	recipes, err := s.recipes.ListDetailed(ctx)
	if err != nil {
		return suggestion.WeekPlanResult{}, err
	}
	stocks, err := s.stock.List(ctx)
	if err != nil {
		return suggestion.WeekPlanResult{}, err
	}
	today := s.now()
	history, err := s.history.ListRecipeHistory(ctx, today.Format(fridge.DateLayout))
	if err != nil {
		return suggestion.WeekPlanResult{}, err
	}

	// 予定のあるレシピは「使い切り」の候補でも使わない。二重の割り当てを誘うため。
	upcoming := map[string]bool{}
	lastCooked := map[string]string{}
	for _, h := range history {
		upcoming[h.RecipeID] = h.HasUpcomingPlan
		lastCooked[h.RecipeID] = h.LastCookedDate
	}
	usable := make([]recipe.Recipe, 0, len(recipes))
	for _, r := range recipes {
		if !upcoming[r.ID] {
			usable = append(usable, r)
		}
	}
	// 候補の数は SuggestionLimit で切らず、条件を満たすレシピをすべて使う (design.md D3)。
	byStock := suggestion.ByStock(usable, stocks, today, cfg.ExpiryAlertRedDays, cfg.ExpiryAlertYellowDays, -1)
	byRecency := suggestion.ByRecency(usable, history, today, cfg.RecencyThresholdDays, -1)
	rest := append([]recipe.Recipe(nil), usable...)
	sort.SliceStable(rest, func(i, j int) bool {
		a, b := lastCooked[rest[i].ID], lastCooked[rest[j].ID]
		if a != b {
			// 一度も作っていないレシピ (空) は最後に回す。
			if a == "" || b == "" {
				return b == ""
			}
			return a < b
		}
		return rest[i].Name < rest[j].Name
	})

	result.Plans = suggestion.WeekPlans(suggestion.SplitSlots(slots, cfg.WeeklyPresetMaxRunDays), byStock, byRecency, rest)
	return result, nil
}

// emptySlots は S09 の曜日と時間帯に当たるセルのうち、品が 1 つも無いセルを日付の順に返す。品のあるセルは利用者が決めたものとして埋めない。
func (s *SuggestionService) emptySlots(ctx context.Context, weekStart string, days []int, meal string) ([]suggestion.WeekSlot, error) {
	entries, err := s.meals.ListPeriod(ctx, weekStart, weekDays)
	if err != nil {
		return nil, err
	}
	filled := map[string]bool{}
	for _, e := range entries {
		if string(e.Meal) == meal {
			filled[e.Date] = true
		}
	}
	start, _ := time.Parse(fridge.DateLayout, weekStart)
	out := make([]suggestion.WeekSlot, 0)
	for i := 0; i < weekDays; i++ {
		d := start.AddDate(0, 0, i)
		// time.Weekday は日曜 = 0 なので、S09 の月曜 = 1 〜 日曜 = 7 にそろえる。
		wd := int(d.Weekday())
		if wd == 0 {
			wd = 7
		}
		date := d.Format(fridge.DateLayout)
		if slices.Contains(days, wd) && !filled[date] {
			out = append(out, suggestion.WeekSlot{Date: date, Meal: meal})
		}
	}
	return out, nil
}
