import { useTranslation } from "react-i18next";
import { Meal } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { MEAL_RENDER_ORDER } from "./mealplan-config";

type Props = {
  value: Set<string>;
  onChange: (next: Set<string>) => void;
};

const LABEL_KEY: Record<string, string> = {
  [Meal.MealBreakfast]: "pages.meal_plan.meal.breakfast",
  [Meal.MealLunch]: "pages.meal_plan.meal.lunch",
  [Meal.MealDinner]: "pages.meal_plan.meal.dinner",
};

export function MealVisibilityCheckboxes({ value, onChange }: Props) {
  const { t } = useTranslation();

  return (
    <fieldset className="inline-flex items-center gap-3 rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm">
      <legend className="sr-only">{t("pages.meal_plan.meal_visibility.aria_label")}</legend>
      {MEAL_RENDER_ORDER.map((meal) => {
        const checked = value.has(meal);
        // 常に 1 個以上の時間帯を残すため、唯一 ON の checkbox は disable にする。
        const isLastOne = checked && value.size === 1;
        return (
          <label
            key={meal}
            className={`flex items-center gap-1.5 ${
              isLastOne ? "cursor-not-allowed text-neutral-400" : "cursor-pointer text-neutral-800"
            }`}
          >
            <input
              type="checkbox"
              className="h-4 w-4 accent-blue-600"
              checked={checked}
              disabled={isLastOne}
              onChange={(e) => {
                const next = new Set(value);
                if (e.target.checked) {
                  next.add(meal);
                } else {
                  next.delete(meal);
                }
                onChange(next);
              }}
            />
            {t(LABEL_KEY[meal])}
          </label>
        );
      })}
    </fieldset>
  );
}
