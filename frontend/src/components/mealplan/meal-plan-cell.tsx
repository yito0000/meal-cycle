import { useTranslation } from "react-i18next";
import type { Entry } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { MEALPLAN_DEFAULTS } from "./mealplan-config";

type Props = {
  entries: Entry[];
};

export function MealPlanCell({ entries }: Props) {
  const { t } = useTranslation();
  const limit = MEALPLAN_DEFAULTS.mealPlanCellVisibleItems;

  if (entries.length === 0) {
    return <div className="min-h-16 rounded-md border border-dashed border-neutral-200 bg-white" />;
  }

  const sorted = [...entries].sort((a, b) => a.position - b.position);
  const visible = sorted.slice(0, limit);
  const overflow = sorted.length - visible.length;

  return (
    <div className="min-h-16 flex flex-col gap-1 rounded-md border border-neutral-200 bg-white p-2">
      {visible.map((entry) => (
        <div key={entry.position} className="truncate text-sm text-neutral-800">
          {entry.recipeName}
        </div>
      ))}
      {overflow > 0 && (
        <div className="text-xs text-neutral-500">
          {t("pages.meal_plan.cell.more_items", { count: overflow })}
        </div>
      )}
    </div>
  );
}
