import { useTranslation } from "react-i18next";
import type { Dish } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";
import { formatDishDates } from "./format-dish-dates";

type Props = {
  dishes: (Dish & { dates: string[]; meals: string[] })[];
};

// 材料の計上単位 (作り置き × レシピ) と 1 行を揃え、何を何回分として数えたかを見せる (design.md D14)。
export function ShoppingDishList({ dishes }: Props) {
  const { t } = useTranslation();
  return (
    <section
      aria-labelledby="shopping-dishes-title"
      className="rounded-md border border-neutral-200 bg-white p-4"
    >
      <h2 id="shopping-dishes-title" className="mb-3 text-sm font-semibold text-neutral-700">
        {t("pages.shopping.dishes.title")}
      </h2>
      <ul className="flex flex-col gap-2">
        {dishes.map((dish) => (
          <li
            key={`${dish.batchId}\u0000${dish.recipeId}`}
            data-testid="shopping-dish"
            data-cooked={dish.cooked}
            className="flex flex-col gap-0.5 text-sm"
          >
            <span className="flex min-w-0 items-center gap-1 font-medium">
              {dish.batchMemberCount > 1 && (
                <span role="img" aria-label={t("pages.shopping.dishes.batch")} className="text-xs">
                  🔗
                </span>
              )}
              <span className="truncate">{dish.recipeName}</span>
            </span>
            <span className="flex flex-wrap items-center gap-2 text-xs text-neutral-500">
              <span data-testid="shopping-dish-dates">{formatDishDates(dish.dates, t)}</span>
              <span data-testid="shopping-dish-meals">
                {dish.meals.map((m) => t(`pages.shopping.dishes.meal.${m}`)).join(", ")}
              </span>
              {dish.startedBeforePeriod && (
                <span
                  data-testid="shopping-dish-started-before"
                  className="rounded border border-amber-300 bg-amber-50 px-1 py-0.5 leading-none text-amber-900"
                >
                  {t("pages.shopping.dishes.started_before")}
                </span>
              )}
              {dish.cooked && (
                <span
                  data-testid="shopping-dish-cooked"
                  className="rounded border border-emerald-300 bg-emerald-50 px-1 py-0.5 leading-none text-emerald-900"
                >
                  {t("pages.shopping.dishes.cooked")}
                </span>
              )}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}
