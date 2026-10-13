import type { TFunction } from "i18next";
import { formatDisplayDate } from "@/lib/format-date";
import { formatQuantity } from "@/lib/format-quantity";
import {
  type Item,
  StockStatus,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";
import { isPantryItem } from "./pantry";
import { periodEndOf, type ShoppingPeriod } from "./shopping-period";

type TextItem = Pick<Item, "name" | "amount" | "unit" | "stockStatus">;

// 絞り込みとチェックは受け取らない。コピーする内容は画面の表示状態に左右されないため (add-shopping-copy-text design.md D1)。
export function buildShoppingText(
  items: readonly TextItem[],
  period: ShoppingPeriod,
  t: TFunction,
): { text: string; count: number } {
  const tableLines = items
    .filter((it) => !isPantryItem(it) && it.stockStatus !== StockStatus.StockCovered)
    .map((it) =>
      t(
        it.stockStatus === StockStatus.StockPartial
          ? "pages.shopping.copy.item_partial"
          : "pages.shopping.copy.item",
        { name: it.name, quantity: formatQuantity(it.amount, it.unit) },
      ),
    );
  const pantryLines = items
    .filter((it) => isPantryItem(it) && it.stockStatus === StockStatus.StockNone)
    .map((it) => t("pages.shopping.copy.pantry_item", { name: it.name }));

  const lines = [
    t("pages.shopping.copy.heading", {
      start: formatDisplayDate(period.periodStart),
      end: formatDisplayDate(periodEndOf(period.periodStart, period.periodDays)),
    }),
    ...tableLines,
  ];
  if (pantryLines.length > 0) {
    lines.push("", t("pages.shopping.copy.pantry_heading"), ...pantryLines);
  }
  return { text: lines.join("\n"), count: tableLines.length + pantryLines.length };
}
