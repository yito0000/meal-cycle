import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import type { StockItem } from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { stockErrorOf } from "../components/fridge/error-messages";
import { FridgeAddRow } from "../components/fridge/fridge-add-row";
import { FridgeSearchInput } from "../components/fridge/fridge-search-input";
import {
  compareStock,
  DEFAULT_FRIDGE_SORT,
  type FridgeSortKey,
} from "../components/fridge/fridge-sort";
import { FridgeSortSelect } from "../components/fridge/fridge-sort-select";
import { FridgeTable } from "../components/fridge/fridge-table";
import {
  UNDO_TOAST_MS,
  useDeleteStock,
  useRestoreStock,
  useStockList,
} from "../components/fridge/use-fridge";
import { useIngredientSuggestions } from "../components/ingredient/use-ingredient-suggestions";

export function FridgePage() {
  const { t } = useTranslation();
  const [search, setSearch] = useState("");
  const [sortKey, setSortKey] = useState<FridgeSortKey>(DEFAULT_FRIDGE_SORT);
  const { data, isLoading, isError } = useStockList();
  const suggestions = useIngredientSuggestions();
  const del = useDeleteStock();
  const restore = useRestoreStock();

  // 最後の 1 行を消すとテーブルがアンマウントされ、mutate に渡したコールバックが呼ばれなく
  // なるため、削除と「元に戻す」は常にマウントされているページ側で扱う。
  const onDelete = (item: StockItem) => {
    del.mutate(item, {
      onSuccess: () => {
        toast(t("pages.fridge.toast.deleted", { name: item.name }), {
          duration: UNDO_TOAST_MS,
          action: {
            label: t("pages.fridge.toast.undo"),
            // 削除前の行をそのまま渡し、同じ ID / 追加日時で作り直す (design.md D8)。
            onClick: () =>
              restore.mutate(item, {
                onError: () => toast.error(t("pages.fridge.toast.restore_failed")),
              }),
          },
        });
      },
      onError: (err) => toast.error(t(stockErrorOf(err).key)),
    });
  };

  const visible = useMemo(() => {
    if (!data) return [];
    const q = search.trim().toLowerCase();
    const filtered = q ? data.filter((s) => s.name.toLowerCase().includes(q)) : data;
    return [...filtered].sort((a, b) => compareStock(a, b, sortKey));
  }, [data, search, sortKey]);

  return (
    <div className="flex h-full flex-col gap-4 p-6">
      <h1 className="text-2xl font-bold">{t("pages.fridge.title")}</h1>

      <div className="flex items-center justify-between gap-4">
        <FridgeSearchInput value={search} onChange={setSearch} />
        <FridgeSortSelect value={sortKey} onChange={setSortKey} />
      </div>

      <FridgeAddRow
        nameSuggestions={suggestions.data?.names ?? []}
        unitSuggestions={suggestions.data?.units ?? []}
      />

      {isLoading && <p className="text-sm text-neutral-500">{t("pages.fridge.loading")}</p>}
      {isError && <p className="text-sm text-red-700">{t("pages.fridge.load_error")}</p>}
      {data && data.length === 0 && (
        <p className="text-sm text-neutral-500">{t("pages.fridge.empty.no_items")}</p>
      )}
      {data && data.length > 0 && visible.length === 0 && (
        <p className="text-sm text-neutral-500">{t("pages.fridge.empty.no_match")}</p>
      )}
      {visible.length > 0 && (
        <FridgeTable
          items={visible}
          nameSuggestions={suggestions.data?.names ?? []}
          deleting={del.isPending}
          onDelete={onDelete}
        />
      )}
    </div>
  );
}
