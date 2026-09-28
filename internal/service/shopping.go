package service

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/yito0000/meal-cycle/internal/config"
	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/domain/shopping"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// ShoppingService は S07 買い物リストをフロントに公開する。
type ShoppingService struct {
	repo  *sqlite.ShoppingRepo
	stock *sqlite.StockRepo
	cfg   config.Config
	now   func() time.Time
}

func NewShoppingService(db *sql.DB) *ShoppingService {
	return &ShoppingService{
		repo:  sqlite.NewShoppingRepo(db),
		stock: sqlite.NewStockRepo(db),
		cfg:   config.Defaults(),
		now:   time.Now,
	}
}

// List は periodStart から periodDays 日間の献立で使う材料と、その元になった献立の一覧を返す。
// 「今日」の決定はフロントの責務で、ここは渡された期間をそのまま使う (design.md D8)。
func (s *ShoppingService) List(ctx context.Context, periodStart string, periodDays int) (shopping.List, error) {
	if err := mealplan.ValidateDate(periodStart); err != nil {
		return shopping.List{}, err
	}
	if !slices.Contains(s.cfg.ShoppingPeriodDayPresets, periodDays) {
		return shopping.List{}, shopping.ErrInvalidPeriodDays
	}

	batches, err := s.repo.ListPlannedBatches(ctx, periodStart, periodDays)
	if err != nil {
		return shopping.List{}, err
	}
	stocks, err := s.usableStocks(ctx)
	if err != nil {
		return shopping.List{}, err
	}

	start, _ := time.Parse(fridge.DateLayout, periodStart)
	return shopping.List{
		PeriodStart: periodStart,
		PeriodEnd:   start.AddDate(0, 0, periodDays-1).Format(fridge.DateLayout),
		PeriodDays:  periodDays,
		Dishes:      shopping.Dishes(batches),
		Items:       shopping.Aggregate(batches, stocks, s.cfg.MeasuringUnits),
	}, nil
}

// 期限切れの在庫は使えない前提で突き合わせから外す。当日期限は urgent なので残る (design.md D6)。
func (s *ShoppingService) usableStocks(ctx context.Context) ([]shopping.StockAmount, error) {
	items, err := s.stock.List(ctx)
	if err != nil {
		return nil, err
	}
	today := s.now()
	out := make([]shopping.StockAmount, 0, len(items))
	for _, it := range items {
		status, _ := fridge.Evaluate(it.ExpiresAt, today, s.cfg.ExpiryAlertRedDays, s.cfg.ExpiryAlertYellowDays)
		if status == fridge.ExpiryExpired {
			continue
		}
		out = append(out, shopping.StockAmount{Name: it.Name, Amount: it.Amount, Unit: it.Unit})
	}
	return out, nil
}
