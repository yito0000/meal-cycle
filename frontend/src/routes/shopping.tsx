import { useNavigate, useSearch } from "@tanstack/react-router";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { formatDisplayDate } from "@/lib/format-date";
import { ShoppingDishList } from "../components/shopping/shopping-dish-list";
import {
  periodEndOf,
  resolveShoppingPeriod,
  type ShoppingPeriod,
} from "../components/shopping/shopping-period";
import { ShoppingPeriodPicker } from "../components/shopping/shopping-period-picker";
import { ShoppingTable } from "../components/shopping/shopping-table";
import { useShoppingList } from "../components/shopping/use-shopping-list";

export function ShoppingPage() {
  const { t } = useTranslation();
  const search = useSearch({ from: "/shopping" });
  const navigate = useNavigate({ from: "/shopping" });
  // 「今日」はマウント時点で固定する。深夜 0 時をまたいでも、開き直すまでは同じ期間を見せる。
  const period = useMemo(() => resolveShoppingPeriod(search, new Date()), [search]);
  const { data, isLoading, isError } = useShoppingList(period.periodStart, period.periodDays);

  const onPeriodChange = (next: ShoppingPeriod) => {
    navigate({
      search: { from: next.periodStart, days: next.periodDays },
      replace: true,
    });
  };

  return (
    <div className="flex h-full flex-col gap-4 p-6">
      <header className="flex flex-col gap-1">
        <h1 className="text-2xl font-bold">{t("pages.shopping.title")}</h1>
        <p className="text-sm text-neutral-500">{t("pages.shopping.description")}</p>
      </header>

      <ShoppingPeriodPicker value={period} onChange={onPeriodChange} />
      <p className="text-sm text-neutral-700" data-testid="shopping-period">
        {t("pages.shopping.period", {
          start: formatDisplayDate(period.periodStart),
          end: formatDisplayDate(periodEndOf(period.periodStart, period.periodDays)),
        })}
      </p>

      {isLoading && <p className="text-sm text-neutral-500">{t("pages.shopping.loading")}</p>}
      {isError && <p className="text-sm text-red-700">{t("pages.shopping.load_error")}</p>}
      {data && data.dishes.length === 0 && (
        <p className="text-sm text-neutral-500">{t("pages.shopping.empty.no_plans")}</p>
      )}
      {data && data.dishes.length > 0 && (
        // 広い画面では材料と献立を左右に並べ、狭い画面では献立を材料の下に回す (design.md D14)。
        <div className="grid grid-cols-1 items-start gap-6 lg:grid-cols-[1fr_20rem]">
          <div>
            {data.items.length === 0 ? (
              <p className="text-sm text-neutral-500">{t("pages.shopping.empty.no_ingredients")}</p>
            ) : (
              <ShoppingTable items={data.items} />
            )}
          </div>
          <ShoppingDishList dishes={data.dishes} />
        </div>
      )}
    </div>
  );
}
