import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import {
  type Item,
  StockStatus,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";
import { itemKey } from "./purchase-form";
import { StockStatusBadge } from "./stock-status-badge";

type Props = {
  items: (Item & { recipes: string[] })[];
};

// 調味料は S05 に登録しないことが多く「在庫なし」の印は情報にならないので、在庫ありだけ印を出す
// (add-shopping-pantry-section design.md D2)。
export function ShoppingPantryList({ items }: Props) {
  const { t } = useTranslation();
  return (
    <section
      aria-labelledby="shopping-pantry-title"
      className="rounded-md border border-neutral-200 bg-white p-4"
    >
      <h2 id="shopping-pantry-title" className="text-sm font-semibold text-neutral-700">
        {t("pages.shopping.pantry.title")}
      </h2>
      <p className="mb-3 text-xs text-neutral-500">{t("pages.shopping.pantry.description")}</p>
      <ul className="flex flex-col gap-2">
        {items.map((item) => {
          const covered = item.stockStatus === StockStatus.StockCovered;
          return (
            <li
              key={itemKey(item)}
              data-testid="shopping-pantry-item"
              data-stock-status={item.stockStatus}
              className={cn(
                "flex flex-wrap items-center gap-x-3 gap-y-0.5 text-sm",
                covered && "text-neutral-400",
              )}
            >
              <span className="font-medium">{item.name}</span>
              <span className={cn("text-xs", covered ? "text-neutral-400" : "text-neutral-500")}>
                {item.recipes.join(", ")}
              </span>
              {covered && <StockStatusBadge status={item.stockStatus} />}
            </li>
          );
        })}
      </ul>
    </section>
  );
}
