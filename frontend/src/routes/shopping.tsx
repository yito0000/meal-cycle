import { useNavigate, useSearch } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { useSettingsValues } from "@/components/settings/settings-gate";
import { Button } from "@/components/ui/button";
import { formatDisplayDate } from "@/lib/format-date";
import { UNDO_TOAST_MS } from "@/lib/undo-toast";
import { StapleState } from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import type { StapleItem } from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";
import { stapleErrorOf } from "../components/fridge/error-messages";
import { useSetStapleState } from "../components/fridge/use-staples";
import { CopyShoppingTextButton } from "../components/shopping/copy-shopping-text-button";
import { isPantryItem } from "../components/shopping/pantry";
import { PurchaseDialog } from "../components/shopping/purchase-dialog";
import { itemKey } from "../components/shopping/purchase-form";
import { ShoppingDishList } from "../components/shopping/shopping-dish-list";
import { ShoppingPantryList } from "../components/shopping/shopping-pantry-list";
import {
  periodEndOf,
  resolveShoppingPeriod,
  type ShoppingPeriod,
} from "../components/shopping/shopping-period";
import { ShoppingPeriodPicker } from "../components/shopping/shopping-period-picker";
import { ShoppingStapleList } from "../components/shopping/shopping-staple-list";
import { ShoppingStockFilter } from "../components/shopping/shopping-stock-filter";
import { ShoppingTable } from "../components/shopping/shopping-table";
import {
  countByStockFilter,
  DEFAULT_STOCK_FILTER,
  matchesStockFilter,
  type StockFilter,
} from "../components/shopping/stock-filter";
import { useShoppingList } from "../components/shopping/use-shopping-list";

export function ShoppingPage() {
  const { t } = useTranslation();
  const search = useSearch({ from: "/shopping" });
  const navigate = useNavigate({ from: "/shopping" });
  // 「今日」はマウント時点で固定する。深夜 0 時をまたいでも、開き直すまでは同じ期間を見せる。
  const { shoppingPeriodDayPresets, shoppingDefaultPeriodDays } = useSettingsValues();
  const period = useMemo(
    () =>
      resolveShoppingPeriod(search, new Date(), {
        dayPresets: shoppingPeriodDayPresets,
        defaultDays: shoppingDefaultPeriodDays,
      }),
    [search, shoppingPeriodDayPresets, shoppingDefaultPeriodDays],
  );
  const { data, isLoading, isError } = useShoppingList(period.periodStart, period.periodDays);

  const [checked, setChecked] = useState<Set<string>>(() => new Set());
  const [dialogOpen, setDialogOpen] = useState(false);
  // 期間が変わったら描画中にチェックを捨てる。effect だと古いチェックが 1 フレーム残る (design.md D3)。
  const periodKey = `${period.periodStart}/${period.periodDays}`;
  const [checkedFor, setCheckedFor] = useState(periodKey);
  if (checkedFor !== periodKey) {
    setCheckedFor(periodKey);
    setChecked(new Set());
  }
  // 絞り込みは期間の切替では戻さない。見たい行の種類は期間を変えても同じため (add-shopping-stock-filter D2)。
  const [stockFilter, setStockFilter] = useState<StockFilter>(DEFAULT_STOCK_FILTER);
  const allItems = data?.items ?? [];
  const tableItems = allItems.filter((it) => !isPantryItem(it));
  const pantryItems = allItems.filter(isPantryItem);
  const visibleItems = tableItems.filter((it) => matchesStockFilter(it, stockFilter));
  // 件数とダイアログは隠れた行のチェックも含めるため、絞る前の全行から数える。
  const checkedItems = tableItems.filter((it) => checked.has(itemKey(it)));

  const setStapleState = useSetStapleState();
  const staples = data?.staples ?? [];
  // 「買った」で項目が欄から消えるので、コールバックは常にマウントされているページ側で扱う (add-fridge-staples design.md D9)。
  const onStapleBought = (item: StapleItem) => {
    setStapleState.mutate(
      { id: item.id, state: StapleState.StapleEnough },
      {
        onSuccess: () => {
          toast(t("pages.shopping.staples.bought", { name: item.name }), {
            duration: UNDO_TOAST_MS,
            action: {
              label: t("pages.shopping.staples.undo"),
              onClick: () =>
                setStapleState.mutate(
                  { id: item.id, state: item.state },
                  { onError: (err) => toast.error(t(stapleErrorOf(err))) },
                ),
            },
          });
        },
        onError: (err) => toast.error(t(stapleErrorOf(err))),
      },
    );
  };
  const stapleList = staples.length > 0 && (
    <ShoppingStapleList
      items={staples}
      pending={setStapleState.isPending}
      onBought={onStapleBought}
    />
  );

  const onRegistered = (count: number) => {
    setDialogOpen(false);
    setChecked(new Set());
    toast(t("pages.shopping.purchase.registered", { count }));
  };

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
      <div className="flex min-h-8 items-center justify-between gap-3">
        <p className="text-sm text-neutral-700" data-testid="shopping-period">
          {t("pages.shopping.period", {
            start: formatDisplayDate(period.periodStart),
            end: formatDisplayDate(periodEndOf(period.periodStart, period.periodDays)),
          })}
        </p>
        {data && ((data.dishes.length > 0 && data.items.length > 0) || staples.length > 0) && (
          <CopyShoppingTextButton items={data.items} staples={staples} period={period} />
        )}
      </div>

      {isLoading && <p className="text-sm text-neutral-500">{t("pages.shopping.loading")}</p>}
      {isError && <p className="text-sm text-red-700">{t("pages.shopping.load_error")}</p>}
      {data && data.dishes.length === 0 && (
        <div className="flex flex-col gap-3">
          <p className="text-sm text-neutral-500">{t("pages.shopping.empty.no_plans")}</p>
          {stapleList}
        </div>
      )}
      {data && data.dishes.length > 0 && (
        // 広い画面では材料と献立を左右に並べ、狭い画面では献立を材料の下に回す (design.md D14)。
        <div className="grid grid-cols-1 items-start gap-6 lg:grid-cols-[1fr_20rem]">
          <div>
            {/* すべて作った印付きなら材料も 0 件になるので、材料が無い判定より先に見ないと誤った理由を出す (design.md D5)。 */}
            {data.dishes.every((d) => d.cooked) ? (
              <div className="flex flex-col gap-3">
                <p className="text-sm text-neutral-500" data-testid="shopping-all-cooked">
                  {t("pages.shopping.empty.all_cooked")}
                </p>
                {stapleList}
              </div>
            ) : data.items.length === 0 ? (
              <div className="flex flex-col gap-3">
                {/* 材料がすべて常備品に寄った期間は、材料が無い期間と文言を分ける (add-fridge-staples design.md D10)。 */}
                {data.hasStapleIngredients ? (
                  <p className="text-sm text-neutral-500" data-testid="shopping-only-staples">
                    {t("pages.shopping.empty.only_staples")}
                  </p>
                ) : (
                  <p className="text-sm text-neutral-500">
                    {t("pages.shopping.empty.no_ingredients")}
                  </p>
                )}
                {stapleList}
              </div>
            ) : (
              <div className="flex flex-col gap-3">
                {tableItems.length === 0 ? (
                  <p className="text-sm text-neutral-500" data-testid="shopping-no-quantified">
                    {t("pages.shopping.empty.no_quantified")}
                  </p>
                ) : (
                  <>
                    <ShoppingStockFilter
                      value={stockFilter}
                      counts={countByStockFilter(tableItems)}
                      onChange={setStockFilter}
                    />
                    {visibleItems.length === 0 ? (
                      <p
                        className="text-sm text-neutral-500"
                        data-testid="shopping-filter-no-match"
                      >
                        {t("pages.shopping.filter.no_match")}
                      </p>
                    ) : (
                      <ShoppingTable
                        items={visibleItems}
                        checked={checked}
                        onCheckedChange={setChecked}
                      />
                    )}
                    <div>
                      <Button
                        type="button"
                        disabled={checkedItems.length === 0}
                        onClick={() => setDialogOpen(true)}
                      >
                        {t("pages.shopping.purchase.register", { count: checkedItems.length })}
                      </Button>
                    </div>
                  </>
                )}
                {stapleList}
                {pantryItems.length > 0 && <ShoppingPantryList items={pantryItems} />}
              </div>
            )}
          </div>
          <ShoppingDishList dishes={data.dishes} />
        </div>
      )}
      <PurchaseDialog
        open={dialogOpen}
        items={checkedItems}
        onOpenChange={setDialogOpen}
        onRegistered={onRegistered}
      />
    </div>
  );
}
