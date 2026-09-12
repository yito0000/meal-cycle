import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { MealPlanService } from "../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { MealVisibilityCheckboxes } from "../components/mealplan/meal-visibility-checkboxes";
import { MEALPLAN_DEFAULTS } from "../components/mealplan/mealplan-config";
import { WeekCalendar } from "../components/mealplan/week-calendar";
import { formatDate, weekStartOf } from "../components/mealplan/week-utils";

export function MealPlanPage() {
  const { t } = useTranslation();
  const [visibleMeals, setVisibleMeals] = useState<Set<string>>(
    () => new Set(MEALPLAN_DEFAULTS.defaultVisibleMeals),
  );

  const weekStart = useMemo(() => weekStartOf(new Date()), []);
  const weekStartKey = formatDate(weekStart);

  const { data, isLoading, isError } = useQuery({
    queryKey: ["mealplan", "week", weekStartKey],
    queryFn: () => MealPlanService.ListWeek(weekStartKey),
  });

  return (
    <div className="flex h-full flex-col gap-4 p-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t("pages.meal_plan.title")}</h1>
        <MealVisibilityCheckboxes value={visibleMeals} onChange={setVisibleMeals} />
      </header>

      {isLoading && <p className="text-sm text-neutral-500">{t("pages.meal_plan.loading")}</p>}
      {isError && <p className="text-sm text-red-700">{t("pages.meal_plan.load_error")}</p>}
      {data && <WeekCalendar weekStart={weekStart} entries={data} visibleMeals={visibleMeals} />}
    </div>
  );
}
