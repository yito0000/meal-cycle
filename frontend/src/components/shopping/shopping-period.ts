import { z } from "zod";
import { formatDate } from "../mealplan/week-utils";
import { SHOPPING_DEFAULTS } from "./shopping-config";

// 不正な値で router がエラー画面に落ちないよう、型が合わない項目は undefined に倒して既定期間へ流す。
export const shoppingSearchSchema = z.object({
  from: z.string().optional().catch(undefined),
  days: z.number().int().optional().catch(undefined),
});

export type ShoppingSearch = z.infer<typeof shoppingSearchSchema>;

export type ShoppingPeriod = { periodStart: string; periodDays: number };

export function isPresetDays(days: number): boolean {
  return (SHOPPING_DEFAULTS.periodDayPresets as readonly number[]).includes(days);
}

// "YYYY-MM-DD" が実在する日付か。new Date の繰り上がり (2026-02-30 → 3/2) を往復比較で弾く。
function isValidDate(s: string): boolean {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
  if (!m) return false;
  const d = new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]));
  return formatDate(d) === s;
}

export function resolveShoppingPeriod(search: ShoppingSearch, today: Date): ShoppingPeriod {
  const { from, days } = search;
  if (from !== undefined && days !== undefined && isValidDate(from) && isPresetDays(days)) {
    return { periodStart: from, periodDays: days };
  }
  return { periodStart: formatDate(today), periodDays: SHOPPING_DEFAULTS.defaultPeriodDays };
}

// 期間に含まれる最後の日。ローカル日付で数え、DST をまたいでも日付だけを進める。
export function periodEndOf(periodStart: string, periodDays: number): string {
  const [y, m, d] = periodStart.split("-").map(Number);
  return formatDate(new Date(y, m - 1, d + periodDays - 1));
}
