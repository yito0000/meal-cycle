import { formatAmount } from "@/lib/format-amount";
import type { Item } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";
import {
  type StockFormErrors,
  type StockFormValues,
  validateStockForm,
} from "../fridge/stock-form";

// 材料の行は「食材名 + 単位」で一意 (shopping spec「同一食材の合算」)。
export function itemKey(item: Pick<Item, "name" | "unit">): string {
  return `${item.name}\u0000${item.unit}`;
}

export function toPurchaseRows(items: Item[]): StockFormValues[] {
  return items.map((it) => ({
    name: it.name,
    amount: formatAmount(it.amount),
    unit: it.unit,
    expiresAt: "",
  }));
}

export function validatePurchaseRows(rows: StockFormValues[]): {
  errors: StockFormErrors[];
  hasError: boolean;
} {
  const errors = rows.map(validateStockForm);
  return { errors, hasError: errors.some((e) => Object.keys(e).length > 0) };
}
