import type { TFunction } from "i18next";
import { formatMealDate } from "@/lib/format-meal-date";
import type { Entry } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";

type TextEntry = Pick<Entry, "date" | "meal" | "position" | "recipeName">;

const MEAL_ORDER = ["breakfast", "lunch", "dinner"] as const;

// 時間帯表示と作った印は受け取らない。家族に渡す献立から予定が黙って抜け落ちないよう、全時間帯を含める (add-mealplan-copy-text design.md D2)。
export function buildMealPlanText(
  entries: readonly TextEntry[],
  period: { start: string; end: string },
  t: TFunction,
  language: string,
): { text: string; dayCount: number } {
  const cells = new Map<string, TextEntry[]>();
  for (const e of entries) {
    const key = `${e.date}|${e.meal}`;
    const list = cells.get(key) ?? [];
    list.push(e);
    cells.set(key, list);
  }
  const keys = [...cells.keys()].sort((a, b) => {
    const [da, ma] = a.split("|");
    const [db, mb] = b.split("|");
    if (da !== db) return da < db ? -1 : 1;
    return (
      MEAL_ORDER.indexOf(ma as (typeof MEAL_ORDER)[number]) -
      MEAL_ORDER.indexOf(mb as (typeof MEAL_ORDER)[number])
    );
  });

  const lines = keys.map((key) => {
    const [date, meal] = key.split("|");
    const names = (cells.get(key) ?? [])
      .slice()
      .sort((a, b) => a.position - b.position)
      .map((e) => e.recipeName);
    return t("pages.meal_plan.text_copy.line", {
      date: formatMealDate(date, language),
      meal: t(`pages.meal_plan.text_copy.meal_${meal}`),
      items: names.join(t("pages.meal_plan.text_copy.separator")),
    });
  });

  const heading = t("pages.meal_plan.text_copy.heading", {
    start: formatMealDate(period.start, language),
    end: formatMealDate(period.end, language),
  });
  const dayCount = new Set(entries.map((e) => e.date)).size;
  return { text: [heading, ...lines].join("\n"), dayCount };
}
