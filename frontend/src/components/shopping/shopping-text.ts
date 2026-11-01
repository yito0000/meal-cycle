import type { TFunction } from "i18next";
import { formatDisplayDate } from "@/lib/format-date";
import { formatQuantity } from "@/lib/format-quantity";
import { StapleState } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import {
  type Item,
  type StapleItem,
  StockStatus,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";
import { isPantryItem } from "./pantry";
import { periodEndOf, type ShoppingPeriod } from "./shopping-period";

type TextItem = Pick<Item, "name" | "amount" | "unit" | "stockStatus">;
type TextStaple = Pick<StapleItem, "name" | "state">;

// 絞り込みとチェックは受け取らない。コピーする内容は画面の表示状態に左右されないため (add-shopping-copy-text design.md D1)。
export function buildShoppingText(
  items: readonly TextItem[],
  period: ShoppingPeriod,
  t: TFunction,
  measuringUnits: readonly string[],
  staples: readonly TextStaple[] = [],
): { text: string; count: number } {
  const tableLines = items
    .filter((it) => !isPantryItem(it) && it.stockStatus !== StockStatus.StockCovered)
    .map((it) =>
      t(
        it.stockStatus === StockStatus.StockPartial
          ? "pages.shopping.copy.item_partial"
          : "pages.shopping.copy.item",
        { name: it.name, quantity: formatQuantity(it.amount, it.unit, measuringUnits) },
      ),
    );
  const pantryLines = items
    .filter((it) => isPantryItem(it) && it.stockStatus === StockStatus.StockNone)
    .map((it) => t("pages.shopping.copy.pantry_item", { name: it.name }));
  // 「ない」には印を添えない。材料の「在庫なし」と同じく、買う必要がある状態を既定として読ませる (add-fridge-staples design.md D11)。
  const stapleLines = staples.map((s) =>
    t(
      s.state === StapleState.StapleLow
        ? "pages.shopping.copy.staple_item_low"
        : "pages.shopping.copy.staple_item",
      { name: s.name },
    ),
  );

  const lines = [
    t("pages.shopping.copy.heading", {
      start: formatDisplayDate(period.periodStart),
      end: formatDisplayDate(periodEndOf(period.periodStart, period.periodDays)),
    }),
    ...tableLines,
  ];
  if (stapleLines.length > 0) {
    lines.push("", t("pages.shopping.copy.staple_heading"), ...stapleLines);
  }
  if (pantryLines.length > 0) {
    lines.push("", t("pages.shopping.copy.pantry_heading"), ...pantryLines);
  }
  return {
    text: lines.join("\n"),
    count: tableLines.length + stapleLines.length + pantryLines.length,
  };
}
