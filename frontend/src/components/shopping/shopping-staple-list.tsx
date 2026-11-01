import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { StapleState } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import type { StapleItem } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";

type Props = {
  items: (StapleItem & { recipes: string[] })[];
  pending: boolean;
  onBought: (item: StapleItem) => void;
};

// 購入済チェックは置かない。常備品は量を持たないので一括登録に乗せず、「買った」だけで戻す (add-fridge-staples proposal)。
export function ShoppingStapleList({ items, pending, onBought }: Props) {
  const { t } = useTranslation();
  return (
    <section
      aria-labelledby="shopping-staple-title"
      className="rounded-md border border-neutral-200 bg-white p-4"
    >
      <h2 id="shopping-staple-title" className="text-sm font-semibold text-neutral-700">
        {t("pages.shopping.staples.title")}
      </h2>
      <p className="mb-3 text-xs text-neutral-500">{t("pages.shopping.staples.description")}</p>
      <ul className="flex flex-col gap-2">
        {items.map((item) => {
          const out = item.state === StapleState.StapleOut;
          return (
            <li
              key={item.id}
              data-testid="shopping-staple-item"
              data-state={item.state}
              className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-sm"
            >
              <span className="font-medium">{item.name}</span>
              <span
                className={cn(
                  "inline-flex shrink-0 items-center rounded border px-1.5 py-0.5 text-xs leading-none",
                  out
                    ? "border-red-300 bg-red-50 text-red-800"
                    : "border-amber-300 bg-amber-50 text-amber-900",
                )}
              >
                {t(`pages.fridge.staples.state.${out ? "out" : "low"}`)}
              </span>
              {item.recipes.length > 0 && (
                <span className="text-xs text-neutral-500">
                  {t("pages.shopping.staples.recipes", { recipes: item.recipes.join(", ") })}
                </span>
              )}
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="ml-auto"
                disabled={pending}
                onClick={() => onBought(item)}
              >
                {t("pages.shopping.staples.bought_button")}
              </Button>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
