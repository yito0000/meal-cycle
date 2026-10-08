package service

import (
	"context"
	"errors"

	"github.com/yito0000/meal-cycle/internal/config"
	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// mealPlanRepo は MealPlanService が使う MealPlanRepo の抽象。テストで in-memory 実装に差し替えられる。
type mealPlanRepo interface {
	ListPeriod(ctx context.Context, periodStart string, periodDays int) ([]mealplan.Entry, error)
	Assign(ctx context.Context, date, meal, recipeID string) (mealplan.Entry, error)
	Replace(ctx context.Context, date, meal string, position int, recipeID string) error
	Remove(ctx context.Context, date, meal string, position int) error
	ClearMeal(ctx context.Context, date, meal string) error
	Reorder(ctx context.Context, date, meal string, orderedRecipeIDs []string) error
	CopyLeftover(ctx context.Context, meal string, targets []string,
		sourceEntries []mealplan.Entry, mode mealplan.CopyMode, shareBatch mealplan.ShareBatch) error
	MealEntries(ctx context.Context, date, meal string) ([]mealplan.Entry, error)
	SplitBatch(ctx context.Context, date, meal string, position int) (string, error)
	MergeBatch(ctx context.Context, date, meal string, position int, targetBatchID string) error
	ListMergeCandidates(ctx context.Context, date, meal, recipeID string) ([]mealplan.BatchSummary, error)
	CopyToOtherMeal(ctx context.Context, sourceDate, sourceMeal string, sourcePosition int,
		targetDate, targetMeal string, mode mealplan.CopyMode) error
	MoveEntry(ctx context.Context, sourceDate, sourceMeal string, sourcePosition int,
		targetDate, targetMeal string) error
	SetCooked(ctx context.Context, date, meal string, position int, cooked bool) error
}

// MealPlanService は S01 献立カレンダーのデータをフロントに公開する。
type MealPlanService struct {
	repo mealPlanRepo
	cfg  config.Config
}

func NewMealPlanService(repo *sqlite.MealPlanRepo) *MealPlanService {
	return &MealPlanService{repo: repo, cfg: config.Defaults()}
}

// newMealPlanServiceWithRepo はテスト用に in-memory repo を差し込む入口。
func newMealPlanServiceWithRepo(repo mealPlanRepo) *MealPlanService {
	return &MealPlanService{repo: repo, cfg: config.Defaults()}
}

// ListPeriod は periodStart から periodWeeks 週分 (7 / 14 / 28 日) の Entry を返す。
// periodStart は "YYYY-MM-DD" (任意の日付、月曜スナップは呼び出し側の責任 — design.md D2)。
// periodWeeks は AllowedPeriodWeeks (1 / 2 / 4) 以外を渡すと ErrInvalidPeriodWeeks を返す。
func (s *MealPlanService) ListPeriod(
	ctx context.Context, periodStart string, periodWeeks int,
) ([]mealplan.Entry, error) {
	if err := mealplan.ValidateDate(periodStart); err != nil {
		return nil, err
	}
	if err := mealplan.ValidatePeriodWeeks(periodWeeks); err != nil {
		return nil, err
	}
	return s.repo.ListPeriod(ctx, periodStart, periodWeeks*7)
}

func (s *MealPlanService) AssignRecipe(
	ctx context.Context, date, meal, recipeID string,
) (mealplan.Entry, error) {
	if err := s.validateMealAddress(date, meal); err != nil {
		return mealplan.Entry{}, err
	}
	return s.repo.Assign(ctx, date, meal, recipeID)
}

func (s *MealPlanService) ReplaceEntry(
	ctx context.Context, date, meal string, position int, recipeID string,
) error {
	if err := s.validateMealAddress(date, meal); err != nil {
		return err
	}
	if err := mealplan.ValidatePosition(position); err != nil {
		return err
	}
	return s.repo.Replace(ctx, date, meal, position, recipeID)
}

func (s *MealPlanService) RemoveEntry(
	ctx context.Context, date, meal string, position int,
) error {
	if err := s.validateMealAddress(date, meal); err != nil {
		return err
	}
	if err := mealplan.ValidatePosition(position); err != nil {
		return err
	}
	return s.repo.Remove(ctx, date, meal, position)
}

func (s *MealPlanService) ClearMeal(ctx context.Context, date, meal string) error {
	if err := s.validateMealAddress(date, meal); err != nil {
		return err
	}
	return s.repo.ClearMeal(ctx, date, meal)
}

func (s *MealPlanService) ReorderMeal(
	ctx context.Context, date, meal string, orderedRecipeIDs []string,
) error {
	if err := s.validateMealAddress(date, meal); err != nil {
		return err
	}
	return s.repo.Reorder(ctx, date, meal, orderedRecipeIDs)
}

// CopyLeftover は起点 (date, meal) を含む days 日連続の同 meal にコピーを反映する。
// position < 0 は meal 全体、position >= 0 はその 1 品だけをコピー元とする。
// days は config.LeftoverCopyPresets に含まれる値のみ許容 (call site に閾値を書かない)。
// shareBatch=true (作り置き) は同一レシピの全 target 行が起点行と同じ BatchID を共有する。
// shareBatch=false (別調理) は 1 行ごとに独立した新 BatchID を割り振る (design.md D3)。
func (s *MealPlanService) CopyLeftover(
	ctx context.Context, date, meal string, days int, position int, mode string, shareBatch bool,
) error {
	if err := s.validateMealAddress(date, meal); err != nil {
		return err
	}
	if !s.isAllowedCopyDays(days) {
		return mealplan.ErrInvalidCopyDays
	}
	cm := mealplan.CopyMode(mode)
	if cm != mealplan.CopyModeAppend && cm != mealplan.CopyModeOverwrite {
		return mealplan.ErrInvalidCopyMode
	}

	sourceMeal, err := s.repo.MealEntries(ctx, date, meal)
	if err != nil {
		return err
	}
	var source []mealplan.Entry
	if position < 0 {
		source = sourceMeal
	} else {
		for _, e := range sourceMeal {
			if e.Position == position {
				source = []mealplan.Entry{e}
				break
			}
		}
		if len(source) == 0 {
			return mealplan.ErrEntryNotFound
		}
	}

	targets, err := mealplan.CopyDates(date, days)
	if err != nil {
		return err
	}
	if len(targets) <= 1 {
		// days=1 (起点日のみ) は no-op。起点日に対する重複 append を PlanCopy が弾く。
		return nil
	}
	// 起点日は自身の追記対象から除外 (作り置きは翌日以降に伸ばす目的)。
	filtered := make([]string, 0, len(targets)-1)
	for _, d := range targets {
		if d == date {
			continue
		}
		filtered = append(filtered, d)
	}
	return s.repo.CopyLeftover(ctx, meal, filtered, source, cm, mealplan.ShareBatch(shareBatch))
}

// SplitBatch は 1 行を単独バッチに分離する。新 BatchID を返し、フロントの UI 更新に使う。
func (s *MealPlanService) SplitBatch(
	ctx context.Context, date, meal string, position int,
) (string, error) {
	if err := s.validateMealAddress(date, meal); err != nil {
		return "", err
	}
	if err := mealplan.ValidatePosition(position); err != nil {
		return "", err
	}
	return s.repo.SplitBatch(ctx, date, meal, position)
}

// SetCooked は 1 行が属する (BatchID, RecipeID) の組に作った印を付け外しする。
// 印は同じ組の全行に効き、別の作り置きには広がらない (add-mealplan-cooked-mark design.md D1)。
func (s *MealPlanService) SetCooked(
	ctx context.Context, date, meal string, position int, cooked bool,
) error {
	if err := s.validateMealAddress(date, meal); err != nil {
		return err
	}
	if err := mealplan.ValidatePosition(position); err != nil {
		return err
	}
	return s.repo.SetCooked(ctx, date, meal, position, cooked)
}

// MergeBatch は 1 行の BatchID を targetBatchID に付け替え、既存の作り置きに合流させる。
// targetBatchID を持つ他の行が 1 件も無ければ ErrBatchTargetNotFound、2 条件 (同一レシピ /
// ±LeftoverCopyPresets max 日以内) を満たす行が無ければ ErrBatchMergeDenied。時間帯の一致は
// 要件から外れており、別時間帯の作り置きにも合流できる (proposal: add-mealplan-cross-meal-leftover)。
func (s *MealPlanService) MergeBatch(
	ctx context.Context, date, meal string, position int, targetBatchID string,
) error {
	if err := s.validateMealAddress(date, meal); err != nil {
		return err
	}
	if err := mealplan.ValidatePosition(position); err != nil {
		return err
	}
	if targetBatchID == "" {
		return mealplan.ErrBatchTargetNotFound
	}
	return s.repo.MergeBatch(ctx, date, meal, position, targetBatchID)
}

// ListMergeCandidates は合流候補ダイアログ用に、2 条件 (同一レシピ / ±日以内) を満たす
// 他バッチ代表行を返す。recipeID は起点行のレシピ ID (呼び出し側は Entry.RecipeID から渡す)。
// 候補には別時間帯の作り置きも含まれ、BatchSummary.Meal で識別できる (proposal:
// add-mealplan-cross-meal-leftover)。
func (s *MealPlanService) ListMergeCandidates(
	ctx context.Context, date, meal, recipeID string,
) ([]mealplan.BatchSummary, error) {
	if err := s.validateMealAddress(date, meal); err != nil {
		return nil, err
	}
	if recipeID == "" {
		return nil, mealplan.ErrUnknownRecipe
	}
	return s.repo.ListMergeCandidates(ctx, date, meal, recipeID)
}

// CopyToOtherMeal は起点行 (sourceDate, sourceMeal, sourcePosition) の作り置きを、別時間帯 (targetMeal)
// の targetDate に持ち越す (同一 batch_id + 同一 recipe の新行を挿入する)。
// mode は CopyLeftover と同じ意味で、既存衝突の解決方針を選ぶ ("append" / "overwrite")。
// 同一 meal / 日付レンジ外 / 未知の mode はすべて型付きエラーで弾き、DB に到達しない。
func (s *MealPlanService) CopyToOtherMeal(
	ctx context.Context,
	sourceDate, sourceMeal string, sourcePosition int,
	targetDate, targetMeal, mode string,
) error {
	if err := s.validateMealAddress(sourceDate, sourceMeal); err != nil {
		return err
	}
	if err := s.validateMealAddress(targetDate, targetMeal); err != nil {
		return err
	}
	if err := mealplan.ValidatePosition(sourcePosition); err != nil {
		return err
	}
	cm := mealplan.CopyMode(mode)
	if cm != mealplan.CopyModeAppend && cm != mealplan.CopyModeOverwrite {
		return mealplan.ErrInvalidCopyMode
	}
	return s.repo.CopyToOtherMeal(ctx, sourceDate, sourceMeal, sourcePosition, targetDate, targetMeal, cm)
}

// MoveEntry は 1 行を別セル (targetDate, targetMeal) の末尾に移す (D&D による移動)。
// batch_id / recipe_id は据え置きで、作り置きに属する行を動かしても batch は解体されない。
// 同一セル (source == target) は責務分離のため repo が ErrInvalidPosition を返し、UI 側で
// ReorderMeal に流させる (design.md D2)。
func (s *MealPlanService) MoveEntry(
	ctx context.Context,
	sourceDate, sourceMeal string, sourcePosition int,
	targetDate, targetMeal string,
) error {
	if err := s.validateMealAddress(sourceDate, sourceMeal); err != nil {
		return err
	}
	if err := s.validateMealAddress(targetDate, targetMeal); err != nil {
		return err
	}
	if err := mealplan.ValidatePosition(sourcePosition); err != nil {
		return err
	}
	return s.repo.MoveEntry(ctx, sourceDate, sourceMeal, sourcePosition, targetDate, targetMeal)
}

func (s *MealPlanService) validateMealAddress(date, meal string) error {
	if err := mealplan.ValidateDate(date); err != nil {
		return err
	}
	if err := mealplan.ValidateMeal(meal); err != nil {
		return err
	}
	return nil
}

func (s *MealPlanService) isAllowedCopyDays(days int) bool {
	for _, allowed := range s.cfg.LeftoverCopyPresets {
		if days == allowed {
			return true
		}
	}
	return false
}

// 型付きエラーは Wails 経由でも errors.Is で拾えるように exported される。
var (
	_ = errors.Is
)
