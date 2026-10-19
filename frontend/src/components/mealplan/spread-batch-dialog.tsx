import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Meal } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { MealPlanService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { formatMealDate } from "../../lib/format-meal-date";
import type {
  SpreadConflictMode,
  SpreadSelection,
  SpreadSource,
  SpreadTarget,
} from "../../lib/spread-batch-selection";
import { Button } from "../ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "../ui/dialog";
import { MEAL_RENDER_ORDER } from "./mealplan-config";

interface Props {
  open: boolean;
  source: SpreadSource;
  sourceRecipeName: string;
  sourceRecipeId: string;
  sourceBatchId: string;
  presetDays: number;
  onOpenChange: (open: boolean) => void;
  onConfirm: (selection: SpreadSelection) => void;
}

function mealLabelKey(meal: string): string {
  switch (meal) {
    case Meal.MealBreakfast:
      return "pages.meal_plan.meal.breakfast";
    case Meal.MealLunch:
      return "pages.meal_plan.meal.lunch";
    default:
      return "pages.meal_plan.meal.dinner";
  }
}

function addDaysISO(date: string, days: number): string {
  const [y, m, d] = date.split("-").map(Number);
  const t = new Date(y, m - 1, d + days);
  const yy = t.getFullYear();
  const mm = String(t.getMonth() + 1).padStart(2, "0");
  const dd = String(t.getDate()).padStart(2, "0");
  return `${yy}-${mm}-${dd}`;
}

function targetKey(date: string, meal: string): string {
  return `${date}::${meal}`;
}

// 対象日レンジ = 起点日 + 未来 maxOffsetDays 日 × 時間帯 3 列の格子で複数選択させる統合ダイアログ。
// 過去方向は「作り置きは翌日以降に伸ばす」意図に反するため対象外。
// 起点セル自身、起点日の起点より前の時間帯、起点と同じ作り置きの同じレシピが入っているセルは選択不可。既存割当セルは warning tone で示唆する。
// 格子は S01 の表示期間の外にはみ出すので、判定に使う献立は起点日から自分で取る (revise-mealplan-spread-safety design.md D1)。
// 行数と献立の範囲は Go の検証と同じ上限から SpreadGrid で受け取り、フロントで計算しない (revise-settings-constraints design.md D3)。
export function SpreadBatchDialog({
  open,
  source,
  sourceRecipeName,
  sourceRecipeId,
  sourceBatchId,
  presetDays,
  onOpenChange,
  onConfirm,
}: Props) {
  const { t, i18n } = useTranslation();

  const gridQuery = useQuery({
    queryKey: ["mealplan", "spread-grid", source.id],
    queryFn: () => MealPlanService.SpreadGrid(source.id),
    enabled: open,
  });
  const gridEntries = gridQuery.data?.entries;
  const isLoaded = gridQuery.isSuccess;
  // 読み終えるまで行数が決まらないので、起点日の 1 行だけを描く。
  const maxOffsetDays = gridQuery.data?.maxOffsetDays ?? 0;

  const rowDates = useMemo(() => {
    const dates: string[] = [];
    for (let i = 0; i <= maxOffsetDays; i++) {
      dates.push(addDaysISO(source.date, i));
    }
    return dates;
  }, [source.date, maxOffsetDays]);

  useEffect(() => {
    if (!gridQuery.isError) return;
    onOpenChange(false);
    toast.error(t("pages.meal_plan.spread.load_failed"));
  }, [gridQuery.isError, onOpenChange, t]);

  const occupancy = useMemo(() => {
    const set = new Set<string>();
    for (const e of gridEntries ?? []) {
      set.add(targetKey(e.date, e.meal));
    }
    return set;
  }, [gridEntries]);

  const sameBatchCells = useMemo(() => {
    const set = new Set<string>();
    for (const e of gridEntries ?? []) {
      if (e.batchId === sourceBatchId && e.recipeId === sourceRecipeId) {
        set.add(targetKey(e.date, e.meal));
      }
    }
    return set;
  }, [gridEntries, sourceBatchId, sourceRecipeId]);

  const sourceMealIndex = (MEAL_RENDER_ORDER as readonly string[]).indexOf(source.meal);

  const defaultSelection = useMemo(() => {
    const set = new Set<string>();
    // presetDays は起点日を含む合計日数 (旧「連続 2 日」= 起点 + 1 日ぶん)。
    // ここでは起点日以降 1..presetDays-1 の同 meal をチェック済みにする。
    for (let i = 1; i < presetDays; i++) {
      const key = targetKey(addDaysISO(source.date, i), source.meal);
      if (!sameBatchCells.has(key)) set.add(key);
    }
    return set;
  }, [source.date, source.meal, presetDays, sameBatchCells]);

  const [selected, setSelected] = useState<Set<string>>(defaultSelection);
  const [shareBatch, setShareBatch] = useState(true);
  const [conflictMode, setConflictMode] = useState<SpreadConflictMode>("append");

  useEffect(() => {
    if (open) {
      setSelected(new Set(defaultSelection));
      setShareBatch(true);
      setConflictMode("append");
    }
  }, [open, defaultSelection]);

  const toggle = (date: string, meal: string) => {
    const key = targetKey(date, meal);
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const formatRow = (date: string): string => formatMealDate(date, i18n.language);

  const targets: SpreadTarget[] = useMemo(() => {
    const out: SpreadTarget[] = [];
    for (const key of selected) {
      const [date, meal] = key.split("::");
      out.push({ date, meal });
    }
    return out;
  }, [selected]);

  // 取得前に反映すると、既存割当の背景色を見ないまま上書きを選べてしまう。
  const canSubmit = isLoaded && targets.length > 0;

  const submit = () => {
    if (!canSubmit) return;
    onConfirm({ targets, shareBatch, conflictMode });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("pages.meal_plan.spread.title")}</DialogTitle>
        </DialogHeader>
        <p className="text-xs text-neutral-500">
          {t("pages.meal_plan.spread.origin_hint", {
            date: formatRow(source.date),
            meal: t(mealLabelKey(source.meal)),
            recipeName: sourceRecipeName,
          })}
        </p>
        <div className="max-h-96 overflow-y-auto rounded-md border border-neutral-200">
          <table className="w-full border-collapse text-sm">
            <thead className="sticky top-0 bg-neutral-50">
              <tr>
                <th className="border-b border-neutral-200 px-2 py-1 text-left text-xs font-medium text-neutral-500">
                  {t("pages.meal_plan.spread.date_header")}
                </th>
                {MEAL_RENDER_ORDER.map((m) => (
                  <th
                    key={m}
                    className="border-b border-neutral-200 px-2 py-1 text-center text-xs font-medium text-neutral-500"
                  >
                    {t(mealLabelKey(m))}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rowDates.map((date) => (
                <tr key={date} className="border-b border-neutral-100 last:border-b-0">
                  <td className="px-2 py-1 text-xs text-neutral-700">{formatRow(date)}</td>
                  {MEAL_RENDER_ORDER.map((meal, mealIndex) => {
                    const isSource = date === source.date && meal === source.meal;
                    const isEarlierMeal = date === source.date && mealIndex < sourceMealIndex;
                    const key = targetKey(date, meal);
                    const isSameBatch = !isSource && !isEarlierMeal && sameBatchCells.has(key);
                    const checked = selected.has(key);
                    const isOccupied = occupancy.has(key);
                    const conflictCls =
                      isOccupied && !isSource && !isEarlierMeal && !isSameBatch
                        ? "bg-amber-50"
                        : "";
                    return (
                      <td
                        key={key}
                        className={`px-2 py-1 text-center ${conflictCls}`}
                        data-date={date}
                        data-meal={meal}
                      >
                        {isSource || isEarlierMeal || isSameBatch ? (
                          <span
                            role="img"
                            aria-label={t(
                              isSource
                                ? "pages.meal_plan.spread.source_cell_label"
                                : isEarlierMeal
                                  ? "pages.meal_plan.spread.earlier_meal_cell_label"
                                  : "pages.meal_plan.spread.same_batch_cell_label",
                            )}
                            className="text-neutral-400"
                          >
                            ●
                          </span>
                        ) : (
                          <input
                            type="checkbox"
                            checked={checked}
                            disabled={!isLoaded}
                            onChange={() => toggle(date, meal)}
                            aria-label={t("pages.meal_plan.spread.cell_checkbox_label", {
                              date: formatRow(date),
                              meal: t(mealLabelKey(meal)),
                            })}
                          />
                        )}
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="flex flex-col gap-3 text-sm">
          <fieldset className="flex flex-col gap-1">
            <legend className="text-xs font-medium text-neutral-500">
              {t("pages.meal_plan.spread.mode_legend")}
            </legend>
            <div className="flex gap-2">
              <label
                className={`cursor-pointer rounded-md border px-3 py-1 text-sm ${
                  shareBatch
                    ? "border-blue-500 bg-blue-50 text-blue-800"
                    : "border-neutral-300 bg-white text-neutral-700 hover:bg-neutral-50"
                }`}
              >
                <input
                  type="radio"
                  name="spread-mode"
                  className="sr-only"
                  checked={shareBatch}
                  onChange={() => setShareBatch(true)}
                />
                {t("pages.meal_plan.spread.share_batch_label")}
              </label>
              <label
                className={`cursor-pointer rounded-md border px-3 py-1 text-sm ${
                  !shareBatch
                    ? "border-blue-500 bg-blue-50 text-blue-800"
                    : "border-neutral-300 bg-white text-neutral-700 hover:bg-neutral-50"
                }`}
              >
                <input
                  type="radio"
                  name="spread-mode"
                  className="sr-only"
                  checked={!shareBatch}
                  onChange={() => setShareBatch(false)}
                />
                {t("pages.meal_plan.spread.distinct_label")}
              </label>
            </div>
          </fieldset>
          <fieldset className="flex flex-col gap-1">
            <legend className="text-xs font-medium text-neutral-500">
              {t("pages.meal_plan.spread.conflict_legend")}
            </legend>
            <div className="flex gap-2">
              <label
                className={`cursor-pointer rounded-md border px-3 py-1 text-sm ${
                  conflictMode === "append"
                    ? "border-blue-500 bg-blue-50 text-blue-800"
                    : "border-neutral-300 bg-white text-neutral-700 hover:bg-neutral-50"
                }`}
              >
                <input
                  type="radio"
                  name="spread-conflict"
                  className="sr-only"
                  checked={conflictMode === "append"}
                  onChange={() => setConflictMode("append")}
                />
                {t("pages.meal_plan.spread.conflict_append_label")}
              </label>
              <label
                className={`cursor-pointer rounded-md border px-3 py-1 text-sm ${
                  conflictMode === "overwrite"
                    ? "border-blue-500 bg-blue-50 text-blue-800"
                    : "border-neutral-300 bg-white text-neutral-700 hover:bg-neutral-50"
                }`}
              >
                <input
                  type="radio"
                  name="spread-conflict"
                  className="sr-only"
                  checked={conflictMode === "overwrite"}
                  onChange={() => setConflictMode("overwrite")}
                />
                {t("pages.meal_plan.spread.conflict_overwrite_label")}
              </label>
            </div>
          </fieldset>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("pages.meal_plan.spread.cancel")}
          </Button>
          <Button disabled={!canSubmit} onClick={submit}>
            {t("pages.meal_plan.spread.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
