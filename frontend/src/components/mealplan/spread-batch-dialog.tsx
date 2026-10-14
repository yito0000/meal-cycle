import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { Entry } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { Meal } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
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
  existingEntries: Entry[];
  // 初期表示する未来日数 (起点日を除く)。起点日 + initialFutureDays の合計行が最初に見える。
  initialFutureDays: number;
  // 「もっと表示」1 回で伸ばす未来日数。
  expandStepDays: number;
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

// 対象日レンジ = 起点日 + 未来 futureDays 日 × 時間帯 3 列の格子で複数選択させる統合ダイアログ。
// 過去方向は「作り置きは翌日以降に伸ばす」意図に反するため対象外。行が足りない場合は
// 下部の「もっと表示」で expandStepDays ずつ伸ばす。
// 起点セル自身と、起点と同じ作り置きの同じレシピが入っているセルは選択不可。既存割当セルは warning tone で示唆する。
export function SpreadBatchDialog({
  open,
  source,
  sourceRecipeName,
  sourceRecipeId,
  sourceBatchId,
  existingEntries,
  initialFutureDays,
  expandStepDays,
  presetDays,
  onOpenChange,
  onConfirm,
}: Props) {
  const { t, i18n } = useTranslation();

  const [futureDays, setFutureDays] = useState(initialFutureDays);

  useEffect(() => {
    if (open) setFutureDays(initialFutureDays);
  }, [open, initialFutureDays]);

  const rowDates = useMemo(() => {
    const dates: string[] = [];
    for (let i = 0; i <= futureDays; i++) {
      dates.push(addDaysISO(source.date, i));
    }
    return dates;
  }, [source.date, futureDays]);

  const occupancy = useMemo(() => {
    const set = new Set<string>();
    for (const e of existingEntries) {
      set.add(targetKey(e.date, e.meal));
    }
    return set;
  }, [existingEntries]);

  // 期間外 (翌週) の日は entries に無いので漏れるが、反映側の追加は同じレシピを二重に入れない (design.md D5)。
  const sameBatchCells = useMemo(() => {
    const set = new Set<string>();
    for (const e of existingEntries) {
      if (e.batchId === sourceBatchId && e.recipeId === sourceRecipeId) {
        set.add(targetKey(e.date, e.meal));
      }
    }
    return set;
  }, [existingEntries, sourceBatchId, sourceRecipeId]);

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

  const dateFmt = useMemo(
    () =>
      new Intl.DateTimeFormat(i18n.language, {
        month: "numeric",
        day: "numeric",
        weekday: "short",
      }),
    [i18n.language],
  );
  const formatRow = (date: string): string => {
    const [y, m, d] = date.split("-").map(Number);
    return dateFmt.format(new Date(y, m - 1, d));
  };

  const targets: SpreadTarget[] = useMemo(() => {
    const out: SpreadTarget[] = [];
    for (const key of selected) {
      const [date, meal] = key.split("::");
      out.push({ date, meal });
    }
    return out;
  }, [selected]);

  const canSubmit = targets.length > 0;

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
            date: source.date,
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
                  {MEAL_RENDER_ORDER.map((meal) => {
                    const isSource = date === source.date && meal === source.meal;
                    const key = targetKey(date, meal);
                    const isSameBatch = !isSource && sameBatchCells.has(key);
                    const checked = selected.has(key);
                    const isOccupied = occupancy.has(key);
                    const conflictCls =
                      isOccupied && !isSource && !isSameBatch ? "bg-amber-50" : "";
                    return (
                      <td
                        key={key}
                        className={`px-2 py-1 text-center ${conflictCls}`}
                        data-date={date}
                        data-meal={meal}
                      >
                        {isSource || isSameBatch ? (
                          <span
                            role="img"
                            aria-label={t(
                              isSource
                                ? "pages.meal_plan.spread.source_cell_label"
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
                            onChange={() => toggle(date, meal)}
                            aria-label={t("pages.meal_plan.spread.cell_checkbox_label", {
                              date,
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
        {expandStepDays > 0 && (
          <div className="flex justify-center">
            <button
              type="button"
              onClick={() => setFutureDays((n) => n + expandStepDays)}
              className="rounded-md border border-neutral-300 bg-white px-3 py-1 text-xs text-neutral-700 hover:bg-neutral-50"
            >
              {t("pages.meal_plan.spread.show_more", { count: expandStepDays })}
            </button>
          </div>
        )}
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
