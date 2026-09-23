import type { TFunction } from "i18next";

export function formatDaysLeft(t: TFunction, daysLeft: number | null): string {
  if (daysLeft === null) return t("pages.fridge.days_left.none");
  if (daysLeft === 0) return t("pages.fridge.days_left.today");
  if (daysLeft < 0) return t("pages.fridge.days_left.overdue", { count: -daysLeft });
  return t("pages.fridge.days_left.remaining", { count: daysLeft });
}
