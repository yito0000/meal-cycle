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
	ListWeek(ctx context.Context, weekStart string) ([]mealplan.Entry, error)
	Assign(ctx context.Context, date, slot, recipeID string) (mealplan.Entry, error)
	Replace(ctx context.Context, date, slot string, position int, recipeID string) error
	Remove(ctx context.Context, date, slot string, position int) error
	ClearSlot(ctx context.Context, date, slot string) error
	Reorder(ctx context.Context, date, slot string, orderedRecipeIDs []string) error
	CopyLeftover(ctx context.Context, slot string, targets []string,
		sourceEntries []mealplan.Entry, mode mealplan.CopyMode, shareBatch mealplan.ShareBatch) error
	SlotEntries(ctx context.Context, date, slot string) ([]mealplan.Entry, error)
	SplitBatch(ctx context.Context, date, slot string, position int) (string, error)
	MergeBatch(ctx context.Context, date, slot string, position int, targetBatchID string) error
	ListSlotBatches(ctx context.Context, date, slot, recipeID string) ([]mealplan.BatchSummary, error)
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

func (s *MealPlanService) ListWeek(ctx context.Context, weekStart string) ([]mealplan.Entry, error) {
	if err := mealplan.ValidateDate(weekStart); err != nil {
		return nil, err
	}
	return s.repo.ListWeek(ctx, weekStart)
}

func (s *MealPlanService) AssignRecipe(
	ctx context.Context, date, slot, recipeID string,
) (mealplan.Entry, error) {
	if err := s.validateSlotAddress(date, slot); err != nil {
		return mealplan.Entry{}, err
	}
	return s.repo.Assign(ctx, date, slot, recipeID)
}

func (s *MealPlanService) ReplaceEntry(
	ctx context.Context, date, slot string, position int, recipeID string,
) error {
	if err := s.validateSlotAddress(date, slot); err != nil {
		return err
	}
	if err := mealplan.ValidatePosition(position); err != nil {
		return err
	}
	return s.repo.Replace(ctx, date, slot, position, recipeID)
}

func (s *MealPlanService) RemoveEntry(
	ctx context.Context, date, slot string, position int,
) error {
	if err := s.validateSlotAddress(date, slot); err != nil {
		return err
	}
	if err := mealplan.ValidatePosition(position); err != nil {
		return err
	}
	return s.repo.Remove(ctx, date, slot, position)
}

func (s *MealPlanService) ClearSlot(ctx context.Context, date, slot string) error {
	if err := s.validateSlotAddress(date, slot); err != nil {
		return err
	}
	return s.repo.ClearSlot(ctx, date, slot)
}

func (s *MealPlanService) ReorderSlot(
	ctx context.Context, date, slot string, orderedRecipeIDs []string,
) error {
	if err := s.validateSlotAddress(date, slot); err != nil {
		return err
	}
	return s.repo.Reorder(ctx, date, slot, orderedRecipeIDs)
}

// CopyLeftover は起点 (date, slot) を含む days 日連続の同 slot にコピーを反映する。
// position < 0 は slot 全体、position >= 0 はその 1 品だけをコピー元とする。
// days は config.LeftoverCopyPresets に含まれる値のみ許容 (call site に閾値を書かない)。
// shareBatch=true (作り置き) は同一レシピの全 target 行が起点行と同じ BatchID を共有する。
// shareBatch=false (別調理) は 1 行ごとに独立した新 BatchID を割り振る (design.md D3)。
func (s *MealPlanService) CopyLeftover(
	ctx context.Context, date, slot string, days int, position int, mode string, shareBatch bool,
) error {
	if err := s.validateSlotAddress(date, slot); err != nil {
		return err
	}
	if !s.isAllowedCopyDays(days) {
		return mealplan.ErrInvalidCopyDays
	}
	cm := mealplan.CopyMode(mode)
	if cm != mealplan.CopyModeAppend && cm != mealplan.CopyModeOverwrite {
		return mealplan.ErrInvalidCopyMode
	}

	sourceSlot, err := s.repo.SlotEntries(ctx, date, slot)
	if err != nil {
		return err
	}
	var source []mealplan.Entry
	if position < 0 {
		source = sourceSlot
	} else {
		for _, e := range sourceSlot {
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
	return s.repo.CopyLeftover(ctx, slot, filtered, source, cm, mealplan.ShareBatch(shareBatch))
}

// SplitBatch は 1 行を単独バッチに分離する。新 BatchID を返し、フロントの UI 更新に使う。
func (s *MealPlanService) SplitBatch(
	ctx context.Context, date, slot string, position int,
) (string, error) {
	if err := s.validateSlotAddress(date, slot); err != nil {
		return "", err
	}
	if err := mealplan.ValidatePosition(position); err != nil {
		return "", err
	}
	return s.repo.SplitBatch(ctx, date, slot, position)
}

// MergeBatch は 1 行の BatchID を targetBatchID に付け替え、既存の作り置きに合流させる。
// targetBatchID を持つ他の行が同じ slot に無ければ ErrBatchTargetNotFound、3 条件 (同 slot /
// 同一レシピ / ±LeftoverCopyPresets max 日以内) を満たす行が無ければ ErrBatchMergeDenied。
func (s *MealPlanService) MergeBatch(
	ctx context.Context, date, slot string, position int, targetBatchID string,
) error {
	if err := s.validateSlotAddress(date, slot); err != nil {
		return err
	}
	if err := mealplan.ValidatePosition(position); err != nil {
		return err
	}
	if targetBatchID == "" {
		return mealplan.ErrBatchTargetNotFound
	}
	return s.repo.MergeBatch(ctx, date, slot, position, targetBatchID)
}

// ListSlotBatches は合流候補ダイアログ用に、3 条件 (同 slot / 同一レシピ / ±日以内) を満たす
// 他バッチ代表行を返す。recipeID は起点行のレシピ ID (呼び出し側は Entry.RecipeID から渡す)。
func (s *MealPlanService) ListSlotBatches(
	ctx context.Context, date, slot, recipeID string,
) ([]mealplan.BatchSummary, error) {
	if err := s.validateSlotAddress(date, slot); err != nil {
		return nil, err
	}
	if recipeID == "" {
		return nil, mealplan.ErrUnknownRecipe
	}
	return s.repo.ListSlotBatches(ctx, date, slot, recipeID)
}

func (s *MealPlanService) validateSlotAddress(date, slot string) error {
	if err := mealplan.ValidateDate(date); err != nil {
		return err
	}
	if err := mealplan.ValidateSlot(slot); err != nil {
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
