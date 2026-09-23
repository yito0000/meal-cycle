import type { StockItem } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";

export type FridgeSortKey = "expiry-asc" | "name-asc" | "added-desc";

export const FRIDGE_SORT_KEYS: FridgeSortKey[] = ["expiry-asc", "name-asc", "added-desc"];

export const DEFAULT_FRIDGE_SORT: FridgeSortKey = "expiry-asc";

function compareName(a: StockItem, b: StockItem): number {
  return a.name.localeCompare(b.name, "ja");
}

export function compareStock(a: StockItem, b: StockItem, key: FridgeSortKey): number {
  switch (key) {
    case "expiry-asc": {
      // 期限なし ("") は常に末尾。YYYY-MM-DD は文字列比較で日付順になる。
      if (a.expiresAt !== b.expiresAt) {
        if (a.expiresAt === "") return 1;
        if (b.expiresAt === "") return -1;
        return a.expiresAt < b.expiresAt ? -1 : 1;
      }
      return compareName(a, b);
    }
    case "name-asc":
      return compareName(a, b);
    case "added-desc":
      if (a.addedAt === b.addedAt) return a.id < b.id ? 1 : a.id > b.id ? -1 : 0;
      return a.addedAt < b.addedAt ? 1 : -1;
  }
}
