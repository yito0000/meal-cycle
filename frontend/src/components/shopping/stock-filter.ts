import {
  type Item,
  StockStatus,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";

export type StockFilter = "all" | "needed" | "covered";

export const STOCK_FILTERS: readonly StockFilter[] = ["all", "needed", "covered"];

export const DEFAULT_STOCK_FILTER: StockFilter = "all";

export function matchesStockFilter(item: Pick<Item, "stockStatus">, filter: StockFilter): boolean {
  switch (filter) {
    case "all":
      return true;
    case "needed":
      return (
        item.stockStatus === StockStatus.StockNone || item.stockStatus === StockStatus.StockPartial
      );
    case "covered":
      return item.stockStatus === StockStatus.StockCovered;
  }
}

export function countByStockFilter(
  items: readonly Pick<Item, "stockStatus">[],
): Record<StockFilter, number> {
  const counts: Record<StockFilter, number> = { all: 0, needed: 0, covered: 0 };
  for (const item of items) {
    for (const filter of STOCK_FILTERS) {
      if (matchesStockFilter(item, filter)) counts[filter]++;
    }
  }
  return counts;
}
