package service

import (
	"context"
	"time"

	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// mealPlanRepo は MealPlanService が使う MealPlanRepo の抽象。テストで in-memory 実装に差し替えられる。
type mealPlanRepo interface {
	ListPeriod(ctx context.Context, periodStart string, periodDays int) ([]mealplan.Entry, error)
	Assign(ctx context.Context, date, meal, recipeID string) (mealplan.Entry, error)
	Entry(ctx context.Context, entryID string) (mealplan.Entry, error)
	Replace(ctx context.Context, entryID, recipeID string) error
	Remove(ctx context.Context, entryID string) (mealplan.RemovedEntry, error)
	ClearMeal(ctx context.Context, date, meal string) ([]mealplan.RemovedEntry, error)
	RestoreEntries(ctx context.Context, entries []mealplan.RemovedEntry) error
	Reorder(ctx context.Context, date, meal string, orderedEntryIDs []string) error
	CopyLeftover(ctx context.Context, meal string, targets []string,
		sourceEntries []mealplan.Entry, mode mealplan.CopyMode, shareBatch mealplan.ShareBatch) error
	MealEntries(ctx context.Context, date, meal string) ([]mealplan.Entry, error)
	SplitBatch(ctx context.Context, entryID string) (string, error)
	MergeBatch(ctx context.Context, entryID, targetBatchID string, dayRange int) error
	ListMergeCandidates(ctx context.Context, entryID string, dayRange int) ([]mealplan.BatchSummary, error)
	Spread(ctx context.Context, entryID string,
		targets []mealplan.SpreadTarget, mode mealplan.CopyMode, shareBatch mealplan.ShareBatch) (mealplan.SpreadResult, error)
	UndoSpread(ctx context.Context, result mealplan.SpreadResult) error
	MoveEntry(ctx context.Context, entryID, targetDate, targetMeal string) error
	SetCooked(ctx context.Context, entryID string, cooked bool) error
}

// MealPlanService は S01 献立カレンダーのデータをフロントに公開する。
type MealPlanService struct {
	repo mealPlanRepo
	cfg  configSource
}

func NewMealPlanService(repo *sqlite.MealPlanRepo, cfg configSource) *MealPlanService {
	return &MealPlanService{repo: repo, cfg: cfg}
}

// newMealPlanServiceWithRepo はテスト用に in-memory repo を差し込む入口。
func newMealPlanServiceWithRepo(repo mealPlanRepo, cfg configSource) *MealPlanService {
	return &MealPlanService{repo: repo, cfg: cfg}
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

// ReplaceEntry は 1 品のレシピを差し替える。1 品に対する操作は、並び順ではなく行の ID で指す
// (add-mealplan-entry-id design.md D3)。
func (s *MealPlanService) ReplaceEntry(ctx context.Context, entryID, recipeID string) error {
	if entryID == "" {
		return mealplan.ErrEntryNotFound
	}
	return s.repo.Replace(ctx, entryID, recipeID)
}

// RemoveEntry は 1 品を削除し、「元に戻す」で RestoreEntries に渡すための削除した行の内容を返す。
func (s *MealPlanService) RemoveEntry(ctx context.Context, entryID string) (mealplan.RemovedEntry, error) {
	if entryID == "" {
		return mealplan.RemovedEntry{}, mealplan.ErrEntryNotFound
	}
	return s.repo.Remove(ctx, entryID)
}

// ClearMeal は時間帯の全品を削除し、削除した行を position 昇順で返す。
func (s *MealPlanService) ClearMeal(ctx context.Context, date, meal string) ([]mealplan.RemovedEntry, error) {
	if err := s.validateMealAddress(date, meal); err != nil {
		return nil, err
	}
	return s.repo.ClearMeal(ctx, date, meal)
}

// RestoreEntries は RemoveEntry / ClearMeal で削除した行を元のセルの元の並び順に戻す。
// 中身はフロントを経由して戻ってくるので、壊れた値で行を作らないよう全項目を確かめる
// (revise-mealplan-destructive-actions design.md D2)。
func (s *MealPlanService) RestoreEntries(ctx context.Context, entries []mealplan.RemovedEntry) error {
	if len(entries) == 0 {
		return nil
	}
	if err := s.validateRemovedEntries(entries); err != nil {
		return err
	}
	return s.repo.RestoreEntries(ctx, entries)
}

func (s *MealPlanService) validateRemovedEntries(entries []mealplan.RemovedEntry) error {
	for _, e := range entries {
		if err := s.validateMealAddress(e.Date, string(e.Meal)); err != nil {
			return err
		}
		if err := mealplan.ValidatePosition(e.Position); err != nil {
			return err
		}
		if e.ID == "" || e.RecipeID == "" || e.BatchID == "" {
			return mealplan.ErrInvalidRestore
		}
		if _, err := time.Parse(time.RFC3339, e.CreatedAt); err != nil {
			return mealplan.ErrInvalidRestore
		}
		if e.CookedAt != "" {
			if _, err := time.Parse(time.RFC3339, e.CookedAt); err != nil {
				return mealplan.ErrInvalidRestore
			}
		}
	}
	return nil
}

// ReorderMeal はセル内の品を orderedEntryIDs の順に並べ直す。同じレシピの品が 2 つあっても区別できるよう、
// レシピ ID ではなく行の ID で並びを受け取る。
func (s *MealPlanService) ReorderMeal(
	ctx context.Context, date, meal string, orderedEntryIDs []string,
) error {
	if err := s.validateMealAddress(date, meal); err != nil {
		return err
	}
	return s.repo.Reorder(ctx, date, meal, orderedEntryIDs)
}

// CopyLeftover は起点 (date, meal) の全品を、起点を含む days 日連続の同 meal にコピーする。
// 1 品だけを広げる経路は SpreadEntry に一本化している (add-mealplan-entry-id design.md D4)。
// days は config.LeftoverCopyPresets に含まれる値のみ許容 (call site に閾値を書かない)。
// shareBatch=true (作り置き) は同一レシピの全 target 行が起点行と同じ BatchID を共有する。
// shareBatch=false (別調理) は 1 行ごとに独立した新 BatchID を割り振る (design.md D3)。
func (s *MealPlanService) CopyLeftover(
	ctx context.Context, date, meal string, days int, mode string, shareBatch bool,
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

	source, err := s.repo.MealEntries(ctx, date, meal)
	if err != nil {
		return err
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
func (s *MealPlanService) SplitBatch(ctx context.Context, entryID string) (string, error) {
	if entryID == "" {
		return "", mealplan.ErrEntryNotFound
	}
	return s.repo.SplitBatch(ctx, entryID)
}

// SetCooked は 1 行が属する (BatchID, RecipeID) の組に作った印を付け外しする。
// 印は同じ組の全行に効き、別の作り置きには広がらない (add-mealplan-cooked-mark design.md D1)。
func (s *MealPlanService) SetCooked(ctx context.Context, entryID string, cooked bool) error {
	if entryID == "" {
		return mealplan.ErrEntryNotFound
	}
	return s.repo.SetCooked(ctx, entryID, cooked)
}

// MergeBatch は 1 行の BatchID を targetBatchID に付け替え、既存の作り置きに合流させる。
// targetBatchID を持つ他の行が 1 件も無ければ ErrBatchTargetNotFound、2 条件 (同一レシピ /
// S09 の作り置きの日数の候補の最大値 ±日以内) を満たす行が無ければ ErrBatchMergeDenied。時間帯の一致は
// 要件から外れており、別時間帯の作り置きにも合流できる (proposal: add-mealplan-cross-meal-leftover)。
func (s *MealPlanService) MergeBatch(ctx context.Context, entryID, targetBatchID string) error {
	if entryID == "" {
		return mealplan.ErrEntryNotFound
	}
	if targetBatchID == "" {
		return mealplan.ErrBatchTargetNotFound
	}
	return s.repo.MergeBatch(ctx, entryID, targetBatchID, s.mergeDayRange())
}

// ListMergeCandidates は合流候補ダイアログ用に、2 条件 (同一レシピ / S09 の作り置きの日数の候補の最大値 ±日以内) を満たす
// 他バッチ代表行を返す。レシピは行そのものから読むので、行とレシピの組を取り違えない。
// 候補には別時間帯の作り置きも含まれ、BatchSummary.Meal で識別できる (proposal:
// add-mealplan-cross-meal-leftover)。
func (s *MealPlanService) ListMergeCandidates(ctx context.Context, entryID string) ([]mealplan.BatchSummary, error) {
	if entryID == "" {
		return nil, mealplan.ErrEntryNotFound
	}
	return s.repo.ListMergeCandidates(ctx, entryID, s.mergeDayRange())
}

// SpreadGrid は「作り置きを広げる」の格子に要る、起点から選べる日数とその範囲の献立を返す。
// 範囲を SpreadEntry の検証と同じ maxSpreadOffsetDays で決め、フロントに上限の写しを置かない (revise-settings-constraints design.md D2)。
func (s *MealPlanService) SpreadGrid(ctx context.Context, entryID string) (mealplan.SpreadGrid, error) {
	if entryID == "" {
		return mealplan.SpreadGrid{}, mealplan.ErrEntryNotFound
	}
	src, err := s.repo.Entry(ctx, entryID)
	if err != nil {
		return mealplan.SpreadGrid{}, err
	}
	maxOffset := s.maxSpreadOffsetDays()
	entries, err := s.repo.ListPeriod(ctx, src.Date, maxOffset+1)
	if err != nil {
		return mealplan.SpreadGrid{}, err
	}
	if entries == nil {
		entries = []mealplan.Entry{}
	}
	return mealplan.SpreadGrid{MaxOffsetDays: maxOffset, Entries: entries}, nil
}

// SpreadEntry は起点行の品を、targets で選んだセルだけに 1 回の操作で入れる。
// 1 セルでも反映できなければどのセルにも入れない (revise-mealplan-spread-api design.md D1)。
// shareBatch=true は起点と同じ作り置き、false は各セルを独立した作り置きにする。
// 戻り値は UndoSpread にそのまま渡して取り消すためのもの (revise-mealplan-spread-safety design.md D2)。
func (s *MealPlanService) SpreadEntry(
	ctx context.Context, entryID string,
	targets []mealplan.SpreadTarget, mode string, shareBatch bool,
) (mealplan.SpreadResult, error) {
	if entryID == "" {
		return mealplan.SpreadResult{}, mealplan.ErrEntryNotFound
	}
	cm := mealplan.CopyMode(mode)
	if cm != mealplan.CopyModeAppend && cm != mealplan.CopyModeOverwrite {
		return mealplan.SpreadResult{}, mealplan.ErrInvalidCopyMode
	}
	// 対象セルの範囲は起点のセルから決まるので、起点の住所を先に読む。
	src, err := s.repo.Entry(ctx, entryID)
	if err != nil {
		return mealplan.SpreadResult{}, err
	}
	if err := mealplan.ValidateSpreadTargets(src.Date, src.Meal, targets, s.maxSpreadOffsetDays()); err != nil {
		return mealplan.SpreadResult{}, err
	}
	return s.repo.Spread(ctx, entryID, targets, cm, mealplan.ShareBatch(shareBatch))
}

// UndoSpread は SpreadEntry の結果を受け取り、追加した行を消して上書きで消した行を戻す。
// 中身はフロントを経由して戻ってくるので、RestoreEntries と同じ検証を通す。
func (s *MealPlanService) UndoSpread(ctx context.Context, result mealplan.SpreadResult) error {
	if len(result.AddedEntryIDs) == 0 && len(result.Removed) == 0 {
		return nil
	}
	for _, id := range result.AddedEntryIDs {
		if id == "" {
			return mealplan.ErrInvalidRestore
		}
	}
	if err := s.validateRemovedEntries(result.Removed); err != nil {
		return err
	}
	return s.repo.UndoSpread(ctx, result)
}

// MoveEntry は 1 行を別セル (targetDate, targetMeal) の末尾に移す (D&D による移動)。
// id / batch_id / recipe_id は据え置きで、作り置きに属する行を動かしても batch は解体されない。
// 同一セル (source == target) は責務分離のため repo が ErrInvalidPosition を返し、UI 側で
// ReorderMeal に流させる (design.md D2)。
func (s *MealPlanService) MoveEntry(ctx context.Context, entryID, targetDate, targetMeal string) error {
	if err := s.validateMealAddress(targetDate, targetMeal); err != nil {
		return err
	}
	if entryID == "" {
		return mealplan.ErrEntryNotFound
	}
	return s.repo.MoveEntry(ctx, entryID, targetDate, targetMeal)
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
	for _, allowed := range s.cfg.current().LeftoverCopyPresets {
		if days == allowed {
			return true
		}
	}
	return false
}

// mergeDayRange は「他の作り置きにまとめる」で許容する日付レンジ (±N 日)。
// S09 で保存した値を再起動なしで効かせるため、呼ぶたびに configSource から読む (fix-mealplan-merge-range D1)。
func (s *MealPlanService) mergeDayRange() int {
	max := 0
	for _, d := range s.cfg.current().LeftoverCopyPresets {
		if d > max {
			max = d
		}
	}
	return max
}

// maxSpreadOffsetDays は「作り置きを広げる」の対象日の上限 (起点日から何日後まで)。
// 格子の行数を「連続 N 日コピー」の最大日数にそろえる (起点日を含むので -1)。
func (s *MealPlanService) maxSpreadOffsetDays() int {
	return s.mergeDayRange() - 1
}
