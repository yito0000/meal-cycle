package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/domain/ingredient"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// バリデーション上限。ユーザーが変えたい値ではなく実装上限なので defaults.go には置かない (design.md D11)。
const (
	stockNameMaxRunes = 100
	stockAmountMax    = 99999
	stockUnitMaxRunes = 20
)

// FridgeService は S05 冷蔵庫在庫 / S06 追加・編集をフロントに公開する。
// 献立とレシピのリポジトリは「作った」品の材料から減らす在庫を求めるためだけに読む (add-mealplan-cook-consume design.md D2)。
type FridgeService struct {
	repo    *sqlite.StockRepo
	staples *sqlite.StapleRepo
	meals   *sqlite.MealPlanRepo
	recipes *sqlite.RecipeRepo
	cfg     configSource
	now     func() time.Time
}

func NewFridgeService(db *sql.DB, cfg configSource) *FridgeService {
	return &FridgeService{
		repo:    sqlite.NewStockRepo(db),
		staples: sqlite.NewStapleRepo(db),
		meals:   sqlite.NewMealPlanRepo(db),
		recipes: sqlite.NewRecipeRepo(db),
		cfg:     cfg,
		now:     time.Now,
	}
}

// List は全在庫を返す。各行の ExpiryStatus / DaysLeft は呼び出し時点のローカル日付で算出する。
func (s *FridgeService) List(ctx context.Context) ([]fridge.StockItem, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	today := s.now()
	for i := range items {
		items[i] = s.withExpiry(items[i], today)
	}
	return items, nil
}

// Add は input を検証して 1 行追加する。同名の既存行には合算しない。
func (s *FridgeService) Add(ctx context.Context, input fridge.StockItem) (fridge.StockItem, error) {
	normalized, err := validateStockInput(input)
	if err != nil {
		return fridge.StockItem{}, err
	}
	created, err := s.repo.Create(ctx, normalized)
	if err != nil {
		return fridge.StockItem{}, err
	}
	return s.withExpiry(created, s.now()), nil
}

// AddMany は S07 の一括登録用。途中まで入った状態を見せないため、全行を検証してから 1 トランザクションで入れる (add-shopping-purchase design.md D1)。
func (s *FridgeService) AddMany(ctx context.Context, inputs []fridge.StockItem) ([]fridge.StockItem, error) {
	normalized := make([]fridge.StockItem, 0, len(inputs))
	for i, input := range inputs {
		n, err := validateStockInput(input)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i, err)
		}
		normalized = append(normalized, n)
	}
	created, err := s.repo.CreateMany(ctx, normalized)
	if err != nil {
		return nil, err
	}
	today := s.now()
	for i := range created {
		created[i] = s.withExpiry(created[i], today)
	}
	return created, nil
}

// Update は指定 ID の行を上書きする。AddedAt は変わらない。
func (s *FridgeService) Update(ctx context.Context, id string, input fridge.StockItem) (fridge.StockItem, error) {
	normalized, err := validateStockInput(input)
	if err != nil {
		return fridge.StockItem{}, err
	}
	updated, err := s.repo.Update(ctx, id, normalized)
	if err != nil {
		return fridge.StockItem{}, err
	}
	return s.withExpiry(updated, s.now()), nil
}

func (s *FridgeService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// Restore は削除直後の「元に戻す」用。削除前の行 (ID / AddedAt 込み) をそのまま受け取る。
func (s *FridgeService) Restore(ctx context.Context, item fridge.StockItem) (fridge.StockItem, error) {
	if strings.TrimSpace(item.ID) == "" {
		return fridge.StockItem{}, fridge.ErrStockInvalidRestore
	}
	if _, err := time.Parse(time.RFC3339, item.AddedAt); err != nil {
		return fridge.StockItem{}, fridge.ErrStockInvalidRestore
	}
	normalized, err := validateStockInput(item)
	if err != nil {
		return fridge.StockItem{}, err
	}
	normalized.ID = item.ID
	normalized.AddedAt = item.AddedAt
	restored, err := s.repo.Restore(ctx, normalized)
	if err != nil {
		return fridge.StockItem{}, err
	}
	return s.withExpiry(restored, s.now()), nil
}

func (s *FridgeService) ListSuggestions(ctx context.Context) (fridge.Suggestions, error) {
	return s.repo.ListSuggestions(ctx)
}

// PlanCookConsumption は作った印を付けた品のレシピの材料から、減らす候補の在庫と減らす量の初期値を求める。
func (s *FridgeService) PlanCookConsumption(ctx context.Context, entryID string) (fridge.ConsumptionPlan, error) {
	entry, err := s.meals.Entry(ctx, entryID)
	if err != nil {
		return fridge.ConsumptionPlan{}, err
	}
	rec, err := s.recipes.GetByID(ctx, entry.RecipeID)
	if err != nil {
		return fridge.ConsumptionPlan{}, err
	}
	stocks, err := s.usableStocks(ctx)
	if err != nil {
		return fridge.ConsumptionPlan{}, err
	}
	staples, err := s.staples.List(ctx)
	if err != nil {
		return fridge.ConsumptionPlan{}, err
	}
	names := make([]string, 0, len(staples))
	for _, st := range staples {
		names = append(names, st.Name)
	}
	return fridge.PlanConsumption(rec.Ingredients, stocks, names, s.cfg.current().MeasuringUnits), nil
}

// Consume は複数行の在庫をまとめて減らす。1 件でも不正なら何も変えない (add-mealplan-cook-consume design.md D4)。
func (s *FridgeService) Consume(ctx context.Context, items []fridge.Consumption) error {
	seen := make(map[string]bool, len(items))
	for _, it := range items {
		if it.Amount < 0 {
			return fridge.ErrConsumeAmountOutOfRange
		}
		if seen[it.StockID] {
			return fridge.ErrConsumeDuplicateStock
		}
		seen[it.StockID] = true
	}
	return s.repo.Consume(ctx, items)
}

// 期限切れの在庫は S07 と同じく使えないものとして候補から外す。当日期限は urgent なので残る。
func (s *FridgeService) usableStocks(ctx context.Context) ([]fridge.StockItem, error) {
	items, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]fridge.StockItem, 0, len(items))
	for _, it := range items {
		if it.ExpiryStatus != fridge.ExpiryExpired {
			out = append(out, it)
		}
	}
	return out, nil
}

// ListStaples は全常備品を返す。並び順はフロントが食材名で決める (add-fridge-staples design.md D7)。
func (s *FridgeService) ListStaples(ctx context.Context) ([]fridge.Staple, error) {
	return s.staples.List(ctx)
}

func (s *FridgeService) AddStaple(ctx context.Context, name string) (fridge.Staple, error) {
	n, err := validateStapleName(name)
	if err != nil {
		return fridge.Staple{}, err
	}
	if err := s.checkStapleDuplicate(ctx, n); err != nil {
		return fridge.Staple{}, err
	}
	return s.staples.Create(ctx, n, fridge.StapleEnough)
}

func (s *FridgeService) SetStapleState(ctx context.Context, id string, state fridge.StapleState) (fridge.Staple, error) {
	if !state.Valid() {
		return fridge.Staple{}, fridge.ErrStapleInvalidState
	}
	return s.staples.UpdateState(ctx, id, state)
}

func (s *FridgeService) DeleteStaple(ctx context.Context, id string) error {
	return s.staples.Delete(ctx, id)
}

// RestoreStaple は削除直後の「元に戻す」用。削除のあとに同名が追加されていれば重複として拒む。
func (s *FridgeService) RestoreStaple(ctx context.Context, item fridge.Staple) (fridge.Staple, error) {
	if strings.TrimSpace(item.ID) == "" {
		return fridge.Staple{}, fridge.ErrStapleInvalidRestore
	}
	if _, err := time.Parse(time.RFC3339, item.AddedAt); err != nil {
		return fridge.Staple{}, fridge.ErrStapleInvalidRestore
	}
	if !item.State.Valid() {
		return fridge.Staple{}, fridge.ErrStapleInvalidState
	}
	n, err := validateStapleName(item.Name)
	if err != nil {
		return fridge.Staple{}, err
	}
	if err := s.checkStapleDuplicate(ctx, n); err != nil {
		return fridge.Staple{}, err
	}
	return s.staples.Restore(ctx, fridge.Staple{ID: item.ID, Name: n, State: item.State, AddedAt: item.AddedAt})
}

// S07 の突き合わせと同じそろえ方で比べる。そろえて同じ名前が 2 件あると、材料をどちらに寄せるか決まらなくなる (design.md D3)。
func (s *FridgeService) checkStapleDuplicate(ctx context.Context, name string) error {
	existing, err := s.staples.List(ctx)
	if err != nil {
		return err
	}
	key := ingredient.NormalizeName(name)
	for _, e := range existing {
		if ingredient.NormalizeName(e.Name) == key {
			return fridge.ErrStapleDuplicate
		}
	}
	return nil
}

func validateStapleName(name string) (string, error) {
	n := strings.TrimSpace(name)
	if n == "" {
		return "", fridge.ErrStapleNameEmpty
	}
	if runeLen(n) > stockNameMaxRunes {
		return "", fridge.ErrStapleNameTooLong
	}
	return n, nil
}

func (s *FridgeService) withExpiry(item fridge.StockItem, today time.Time) fridge.StockItem {
	cfg := s.cfg.current()
	item.ExpiryStatus, item.DaysLeft = fridge.Evaluate(
		item.ExpiresAt, today, cfg.ExpiryAlertRedDays, cfg.ExpiryAlertYellowDays,
	)
	return item
}

// validateStockInput は name / unit / expiresAt を trim しつつ制約を検証し、永続化する列だけを返す。
func validateStockInput(input fridge.StockItem) (fridge.StockItem, error) {
	out := fridge.StockItem{
		Name:      strings.TrimSpace(input.Name),
		Amount:    input.Amount,
		Unit:      strings.TrimSpace(input.Unit),
		ExpiresAt: strings.TrimSpace(input.ExpiresAt),
	}
	if out.Name == "" {
		return fridge.StockItem{}, fridge.ErrStockNameEmpty
	}
	if runeLen(out.Name) > stockNameMaxRunes {
		return fridge.StockItem{}, fridge.ErrStockNameTooLong
	}
	if out.Amount <= 0 || out.Amount > stockAmountMax {
		return fridge.StockItem{}, fridge.ErrStockAmountOutOfRange
	}
	if runeLen(out.Unit) > stockUnitMaxRunes {
		return fridge.StockItem{}, fridge.ErrStockUnitTooLong
	}
	if out.ExpiresAt != "" {
		// time.Parse は 2026-02-30 のような実在しない日付も弾く。
		if _, err := time.Parse(fridge.DateLayout, out.ExpiresAt); err != nil {
			return fridge.StockItem{}, fridge.ErrStockInvalidExpiry
		}
	}
	return out, nil
}
