package service

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/yito0000/meal-cycle/internal/config"
	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// バリデーション上限。ユーザーが変えたい値ではなく実装上限なので defaults.go には置かない (design.md D11)。
const (
	stockNameMaxRunes = 100
	stockAmountMax    = 99999
	stockUnitMaxRunes = 20
)

// FridgeService は S05 冷蔵庫在庫 / S06 追加・編集をフロントに公開する。
type FridgeService struct {
	repo *sqlite.StockRepo
	cfg  config.Config
	now  func() time.Time
}

func NewFridgeService(db *sql.DB) *FridgeService {
	return &FridgeService{repo: sqlite.NewStockRepo(db), cfg: config.Defaults(), now: time.Now}
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

func (s *FridgeService) withExpiry(item fridge.StockItem, today time.Time) fridge.StockItem {
	item.ExpiryStatus, item.DaysLeft = fridge.Evaluate(
		item.ExpiresAt, today, s.cfg.ExpiryAlertRedDays, s.cfg.ExpiryAlertYellowDays,
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
