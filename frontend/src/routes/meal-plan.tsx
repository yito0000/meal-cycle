import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { MealPlanService } from "../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { MealVisibilityCheckboxes } from "../components/mealplan/meal-visibility-checkboxes";
import { MEALPLAN_DEFAULTS } from "../components/mealplan/mealplan-config";
import { PeriodNav, type PeriodWeeks } from "../components/mealplan/period-nav";
import { WeekCalendar } from "../components/mealplan/week-calendar";
import { formatDate, weekStartOf } from "../components/mealplan/week-utils";

export function MealPlanPage() {
  const { t } = useTranslation();
  const [visibleMeals, setVisibleMeals] = useState<Set<string>>(
    () => new Set(MEALPLAN_DEFAULTS.defaultVisibleMeals),
  );
  const [periodStart, setPeriodStart] = useState<Date>(() => weekStartOf(new Date()));
  const [periodWeeks, setPeriodWeeks] = useState<PeriodWeeks>(1);

  const periodStartKey = useMemo(() => formatDate(periodStart), [periodStart]);

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
    </div>
  );
}
