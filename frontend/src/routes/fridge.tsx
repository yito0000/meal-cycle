import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { UNDO_TOAST_MS } from "@/lib/undo-toast";
import type {
  Staple,
  StockItem,
} from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { stapleErrorOf, stockErrorOf } from "../components/fridge/error-messages";
import { FridgeAddRow } from "../components/fridge/fridge-add-row";
import { FridgeReviewBar } from "../components/fridge/fridge-review-bar";
import { FridgeSearchInput } from "../components/fridge/fridge-search-input";
import {
  compareStock,
  DEFAULT_FRIDGE_SORT,
  type FridgeSortKey,
} from "../components/fridge/fridge-sort";
import { FridgeSortSelect } from "../components/fridge/fridge-sort-select";
import { FridgeTable } from "../components/fridge/fridge-table";
import { StaplePanel } from "../components/fridge/staple-panel";
import { nextUnreviewedId } from "../components/fridge/stock-review";
import { useDeleteStock, useRestoreStock, useStockList } from "../components/fridge/use-fridge";
import { useDeleteStaple, useRestoreStaple } from "../components/fridge/use-staples";
import { useIngredientSuggestions } from "../components/ingredient/use-ingredient-suggestions";

export function FridgePage() {
  const { t } = useTranslation();
  const [search, setSearch] = useState("");
  const [sortKey, setSortKey] = useState<FridgeSortKey>(DEFAULT_FRIDGE_SORT);
  const { data, isLoading, isError } = useStockList();
  const suggestions = useIngredientSuggestions();
  const del = useDeleteStock();
  const restore = useRestoreStock();
  const delStaple = useDeleteStaple();
  const restoreStaple = useRestoreStaple();
  // 見直し済みの印は保存せず、見直しを終えるか S05 を離れたら捨てる (add-fridge-stock-review design.md D3)。
  const [reviewing, setReviewing] = useState(false);
  const [reviewed, setReviewed] = useState<ReadonlySet<string>>(new Set());
  // undefined はフォーカスを動かさない、id が null なら「見直しを終える」へ、文字列ならその行の「ある」へ動かす。
  const [focusTarget, setFocusTarget] = useState<{ id: string | null } | undefined>();
  const finishRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!focusTarget) return;
    const el =
      focusTarget.id === null
        ? finishRef.current
        : document.querySelector<HTMLButtonElement>(`[data-review-ok="${focusTarget.id}"]`);
    el?.focus();
  }, [focusTarget]);

  // 最後の 1 行を消すとテーブルがアンマウントされ、mutate に渡したコールバックが呼ばれなく
  // なるため、削除と「元に戻す」は常にマウントされているページ側で扱う。
  const onDelete = (item: StockItem, onRestored?: () => void) => {
    del.mutate(item, {
      onSuccess: () => {
        toast(t("pages.fridge.toast.deleted", { name: item.name }), {
          duration: UNDO_TOAST_MS,
          action: {
            label: t("pages.fridge.toast.undo"),
            // 削除前の行をそのまま渡し、同じ ID / 追加日時で作り直す (design.md D8)。
            onClick: () =>
              restore.mutate(item, {
                onSuccess: onRestored,
                onError: () => toast.error(t("pages.fridge.toast.restore_failed")),
              }),
          },
        });
      },
      onError: (err) => toast.error(t(stockErrorOf(err).key)),
    });
  };

  const onDeleteStaple = (item: Staple) => {
    delStaple.mutate(item, {
      onSuccess: () => {
        toast(t("pages.fridge.staples.toast.deleted", { name: item.name }), {
          duration: UNDO_TOAST_MS,
          action: {
            label: t("pages.fridge.toast.undo"),
            onClick: () =>
              restoreStaple.mutate(item, {
                onError: () => toast.error(t("pages.fridge.toast.restore_failed")),
              }),
          },
        });
      },
      onError: (err) => toast.error(t(stapleErrorOf(err))),
    });
  };

  const visible = useMemo(() => {
    if (!data) return [];
    const q = search.trim().toLowerCase();
    const filtered = q ? data.filter((s) => s.name.toLowerCase().includes(q)) : data;
    return [...filtered].sort((a, b) => compareStock(a, b, sortKey));
  }, [data, search, sortKey]);

  const rowIds = visible.map((s) => s.id);
  const reviewedCount = data ? data.filter((s) => reviewed.has(s.id)).length : 0;

  const markReviewed = (item: StockItem) => {
    const next = new Set(reviewed).add(item.id);
    setReviewed(next);
    setFocusTarget({ id: nextUnreviewedId(rowIds, next, item.id) });
  };

  const review = {
    reviewed,
    onOk: markReviewed,
    onReduced: markReviewed,
    onGone: (item: StockItem) => {
      const index = rowIds.indexOf(item.id);
      const remaining = rowIds.filter((id) => id !== item.id);
      onDelete(item, () => {
        // 戻った行はまだ確かめていないので、見直し済みから外す (design.md D4)。
        setReviewed((prev) => {
          const next = new Set(prev);
          next.delete(item.id);
          return next;
        });
      });
      setFocusTarget({ id: nextUnreviewedId(remaining, reviewed, item.id, index) });
    },
  };

  const finishReview = () => {
    setReviewing(false);
    setReviewed(new Set());
    setFocusTarget(undefined);
  };

  return (
    <div className="flex h-full flex-col gap-4 p-6">
      <h1 className="text-2xl font-bold">{t("pages.fridge.title")}</h1>

      <div className="grid grid-cols-1 items-start gap-6 lg:grid-cols-[1fr_20rem]">
        <div className="flex min-w-0 flex-col gap-4">
          <div className="flex items-center justify-between gap-4">
            <FridgeSearchInput value={search} onChange={setSearch} />
            <FridgeSortSelect value={sortKey} onChange={setSortKey} />
          </div>

          <FridgeAddRow
            nameSuggestions={suggestions.data?.names ?? []}
            unitSuggestions={suggestions.data?.units ?? []}
          />

          {data && data.length > 0 && !reviewing && (
            <div>
              <Button type="button" variant="outline" size="sm" onClick={() => setReviewing(true)}>
                {t("pages.fridge.review.start")}
              </Button>
            </div>
          )}
          {reviewing && (
            <FridgeReviewBar
              done={reviewedCount}
              total={data?.length ?? 0}
              onFinish={finishReview}
              finishRef={finishRef}
            />
          )}

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
              key={reviewing ? "review" : "normal"}
              items={visible}
              review={reviewing ? review : undefined}
              nameSuggestions={suggestions.data?.names ?? []}
              deleting={del.isPending}
              onDelete={(item) => onDelete(item)}
            />
          )}
        </div>
        <StaplePanel
          nameSuggestions={suggestions.data?.names ?? []}
          deleting={delStaple.isPending}
          onDelete={onDeleteStaple}
        />
      </div>
    </div>
  );
}
