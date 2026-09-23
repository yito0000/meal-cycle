import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Meal } from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { MealPlanService } from "../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { MealVisibilityCheckboxes } from "../components/mealplan/meal-visibility-checkboxes";
import { MEALPLAN_DEFAULTS } from "../components/mealplan/mealplan-config";
import { PeriodNav, type PeriodWeeks } from "../components/mealplan/period-nav";
import { WeekCalendar } from "../components/mealplan/week-calendar";
import { formatDate, weekStartOf } from "../components/mealplan/week-utils";
import { mealPlanKey, useMealPlanPreference } from "../lib/mealplan-preferences";

const ALLOWED_MEAL_VALUES: ReadonlySet<string> = new Set<string>([
  Meal.MealBreakfast,
  Meal.MealLunch,
  Meal.MealDinner,
]);

const ALLOWED_PERIOD_WEEKS: readonly PeriodWeeks[] = [1, 2, 4];

function validateVisibleMeals(raw: unknown): string[] | null {
  if (!Array.isArray(raw)) return null;
  const next: string[] = [];
  for (const item of raw) {
    if (typeof item !== "string" || !ALLOWED_MEAL_VALUES.has(item)) return null;
    if (!next.includes(item)) next.push(item);
  }
  // 空配列は「常に 1 個以上残す」不変条件を破るため既定値に戻す。
  if (next.length === 0) return null;
  return next;
}

function validatePeriodWeeks(raw: unknown): PeriodWeeks | null {
  return ALLOWED_PERIOD_WEEKS.find((w) => w === raw) ?? null;
}

export function MealPlanPage() {
  const { t } = useTranslation();
  // Set は JSON.stringify で {} になるため、hook 内は string[] で保持し、
  // 呼び出し側で Set に変換する。既存の MealVisibilityCheckboxes API は Set<string> のまま。
  const [visibleMealsArr, setVisibleMealsArr] = useMealPlanPreference<string[]>(
    mealPlanKey("visibleMeals"),
    [...MEALPLAN_DEFAULTS.defaultVisibleMeals],
    validateVisibleMeals,
  );
  const visibleMeals = useMemo(() => new Set<string>(visibleMealsArr), [visibleMealsArr]);
  const setVisibleMeals = useCallback(
    (next: Set<string>) => setVisibleMealsArr([...next]),
    [setVisibleMealsArr],
  );
  const [periodStart, setPeriodStart] = useState<Date>(() => weekStartOf(new Date()));
  const [periodWeeks, setPeriodWeeks] = useMealPlanPreference<PeriodWeeks>(
    mealPlanKey("periodWeeks"),
    1,
    validatePeriodWeeks,
  );

  const periodStartKey = useMemo(() => formatDate(periodStart), [periodStart]);
  const navigate = useNavigate();

  const { data, isLoading, isError } = useQuery({
    queryKey: ["mealplan", "period", periodStartKey, periodWeeks],
    queryFn: () => MealPlanService.ListPeriod(periodStartKey, periodWeeks),
  });

  return (
    <div className="flex h-full flex-col gap-4 p-6">
      <header className="flex items-center justify-between gap-4 flex-wrap">
        <h1 className="text-2xl font-bold">{t("pages.meal_plan.title")}</h1>
        <div className="flex items-center gap-3 flex-wrap">
          <MealVisibilityCheckboxes value={visibleMeals} onChange={setVisibleMeals} />
          <PeriodNav
            periodStart={periodStart}
            periodWeeks={periodWeeks}
            onPeriodChange={(start, weeks) => {
              setPeriodStart(start);
              setPeriodWeeks(weeks);
            }}
            onToday={() => {
              /* PeriodNav 側で periodStart を今週に戻す。ここは追加の副作用が必要な場合のみ使う */
            }}
          />
        </div>
      </header>

      {isLoading && <p className="text-sm text-neutral-500">{t("pages.meal_plan.loading")}</p>}
      {isError && <p className="text-sm text-red-700">{t("pages.meal_plan.load_error")}</p>}
      {data && (
        <WeekCalendar
          periodStart={periodStart}
          periodWeeks={periodWeeks}
          entries={data}
          visibleMeals={visibleMeals}
        />
      )}

      <div>
        <Button
          type="button"
          variant="outline"
          data-testid="shopping-cta"
          onClick={() =>
            // 表示中の時間帯は渡さない。買い物リストは期間内の全時間帯を対象にする (mealplan spec)。
            navigate({
              to: "/shopping",
              search: { from: periodStartKey, days: periodWeeks * 7 },
            })
          }
        >
          {t("pages.meal_plan.shopping_cta")}
        </Button>
      </div>
    </div>
  );
}
