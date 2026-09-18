import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type {
  BatchSummary,
  Entry,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { Meal } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { MealPlanService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { BatchMergeDialog } from "./batch-merge-dialog";
import { CopyConflictDialog, type CopyConflictMode } from "./copy-conflict-dialog";
import { CopyModeDialog } from "./copy-mode-dialog";
import { previewCopy } from "./copy-planner";
import { CopyToMealDialog } from "./copy-to-meal-dialog";
import { errorI18nKey } from "./error-messages";
import { type EntryAction, type MealAction, MealPlanCell } from "./meal-plan-cell";
import { MEAL_RENDER_ORDER, MEALPLAN_DEFAULTS, pickCellPreset } from "./mealplan-config";
import type { PeriodWeeks } from "./period-nav";
import { RecipePickerDialog } from "./recipe-picker-dialog";
import { formatDate, isSameLocalDate, periodDates } from "./week-utils";

type Props = {
  periodStart: Date;
  periodWeeks: PeriodWeeks;
  entries: Entry[];
  visibleMeals: Set<string>;
};

type PickerState =
  | { kind: "closed" }
  | { kind: "add"; date: string; meal: string }
  | { kind: "replace"; date: string; meal: string; position: number };

type PendingCopy = {
  date: string;
  meal: string;
  days: number;
  position: number; // -1 = 時間帯全体
  targets: string[];
  conflicts: string[];
  shareBatch?: boolean;
};

type MergeDialogState =
  | { kind: "closed" }
  | { kind: "open"; date: string; meal: string; position: number; candidates: BatchSummary[] };

type CopyToMealState =
  | { kind: "closed" }
  | { kind: "picking"; sourceDate: string; sourceMeal: string; sourcePosition: number }
  | {
      kind: "confirm-conflict";
      sourceDate: string;
      sourceMeal: string;
      sourcePosition: number;
      targetDate: string;
      targetMeal: string;
    };

// mergeDayRange の source of truth は Go 側 config.LeftoverCopyPresets の max。
// フロントは MEALPLAN_DEFAULTS.leftoverCopyPresets の mirror を使う (S09 で ConfigService 経由に移行)。
const CROSS_MEAL_DAY_RANGE = Math.max(...MEALPLAN_DEFAULTS.leftoverCopyPresets);

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

export function WeekCalendar({ periodStart, periodWeeks, entries, visibleMeals }: Props) {
  const { t, i18n } = useTranslation();
  const qc = useQueryClient();
  const [picker, setPicker] = useState<PickerState>({ kind: "closed" });
  const [pendingCopy, setPendingCopy] = useState<PendingCopy | null>(null);
  const [mergeDialog, setMergeDialog] = useState<MergeDialogState>({ kind: "closed" });
  const [copyToMeal, setCopyToMeal] = useState<CopyToMealState>({ kind: "closed" });
  const [errorKey, setErrorKey] = useState<string | null>(null);

  // 期間切替と前後ナビの再取得は queryKey ["mealplan", "period", …] を共通 prefix ["mealplan"] で
  // 全部落とすように統一する (design.md D4)。
  const invalidate = () => qc.invalidateQueries({ queryKey: ["mealplan"] });

  const withHandlers = <T,>(promise: Promise<T>) =>
    promise
      .then((result) => {
        invalidate();
        setErrorKey(null);
        return result;
      })
      .catch((err) => {
        setErrorKey(errorI18nKey(err));
        throw err;
      });

  const assignMut = useMutation({
    mutationFn: (v: { date: string; meal: string; recipeID: string }) =>
      withHandlers(MealPlanService.AssignRecipe(v.date, v.meal, v.recipeID)),
  });
  const replaceMut = useMutation({
    mutationFn: (v: { date: string; meal: string; position: number; recipeID: string }) =>
      withHandlers(MealPlanService.ReplaceEntry(v.date, v.meal, v.position, v.recipeID)),
  });
  const removeMut = useMutation({
    mutationFn: (v: { date: string; meal: string; position: number }) =>
      withHandlers(MealPlanService.RemoveEntry(v.date, v.meal, v.position)),
  });
  const clearMut = useMutation({
    mutationFn: (v: { date: string; meal: string }) =>
      withHandlers(MealPlanService.ClearMeal(v.date, v.meal)),
  });
  const copyMut = useMutation({
    mutationFn: (v: {
      date: string;
      meal: string;
      days: number;
      position: number;
      mode: string;
      shareBatch: boolean;
    }) =>
      withHandlers(
        MealPlanService.CopyLeftover(v.date, v.meal, v.days, v.position, v.mode, v.shareBatch),
      ),
  });
  const splitMut = useMutation({
    mutationFn: (v: { date: string; meal: string; position: number }) =>
      withHandlers(MealPlanService.SplitBatch(v.date, v.meal, v.position)),
  });
  const mergeMut = useMutation({
    mutationFn: (v: { date: string; meal: string; position: number; targetBatchID: string }) =>
      withHandlers(MealPlanService.MergeBatch(v.date, v.meal, v.position, v.targetBatchID)),
  });
  const copyToMealMut = useMutation({
    mutationFn: (v: {
      sourceDate: string;
      sourceMeal: string;
      sourcePosition: number;
      targetDate: string;
      targetMeal: string;
      mode: string;
    }) =>
      withHandlers(
        MealPlanService.CopyToOtherMeal(
          v.sourceDate,
          v.sourceMeal,
          v.sourcePosition,
          v.targetDate,
          v.targetMeal,
          v.mode,
        ),
      ),
  });

  const dates = periodDates(periodStart, periodWeeks * 7);
  const today = new Date();
  const meals = MEAL_RENDER_ORDER.filter((s) => visibleMeals.has(s));
  const preset = pickCellPreset(periodWeeks, meals.length);
  // 2 週以上は「7 列 × periodWeeks 行」の折り返しレイアウトに揃える (1 週は 1 行のまま)。
  const isWrapped = periodWeeks >= 2;
  const columnsPerBlock = isWrapped ? 7 : periodWeeks * 7;
  const blockCount = isWrapped ? periodWeeks : 1;
  const dateBlocks = Array.from({ length: blockCount }, (_, i) =>
    dates.slice(i * columnsPerBlock, (i + 1) * columnsPerBlock),
  );

  const grouped = new Map<string, Entry[]>();
  for (const e of entries) {
    const key = `${e.date}::${e.meal}`;
    const bucket = grouped.get(key);
    if (bucket) bucket.push(e);
    else grouped.set(key, [e]);
  }

  const startCopy = (date: string, meal: string, days: number, position: number) => {
    const { targets, conflicts } = previewCopy(date, meal, days, entries);
    if (targets.length === 0) return;
    setPendingCopy({ date, meal, days, position, targets, conflicts });
  };

  const resolveMode = (shareBatch: boolean) => {
    if (!pendingCopy) return;
    if (pendingCopy.conflicts.length === 0) {
      copyMut.mutate({
        date: pendingCopy.date,
        meal: pendingCopy.meal,
        days: pendingCopy.days,
        position: pendingCopy.position,
        mode: "append",
        shareBatch,
      });
      setPendingCopy(null);
      return;
    }
    setPendingCopy({ ...pendingCopy, shareBatch });
  };

  const resolveConflict = (mode: CopyConflictMode) => {
    if (!pendingCopy) return;
    if (mode === "cancel") {
      setPendingCopy(null);
      return;
    }
    copyMut.mutate({
      date: pendingCopy.date,
      meal: pendingCopy.meal,
      days: pendingCopy.days,
      position: pendingCopy.position,
      mode,
      shareBatch: pendingCopy.shareBatch ?? true,
    });
    setPendingCopy(null);
  };

  const openMergeDialog = async (
    date: string,
    meal: string,
    position: number,
    recipeID: string,
  ) => {
    try {
      const candidates = (await MealPlanService.ListMergeCandidates(date, meal, recipeID)) ?? [];
      setMergeDialog({ kind: "open", date, meal, position, candidates });
      setErrorKey(null);
    } catch (err) {
      setErrorKey(errorI18nKey(err));
    }
  };

  const submitCopyToMeal = (
    sourceDate: string,
    sourceMeal: string,
    sourcePosition: number,
    targetDate: string,
    targetMeal: string,
    mode: string,
  ) => {
    copyToMealMut.mutate({ sourceDate, sourceMeal, sourcePosition, targetDate, targetMeal, mode });
  };

  const handleCopyToMealPick = (targetDate: string, targetMeal: string) => {
    if (copyToMeal.kind !== "picking") return;
    const { sourceDate, sourceMeal, sourcePosition } = copyToMeal;
    const hasConflict = entries.some((e) => e.date === targetDate && e.meal === targetMeal);
    if (!hasConflict) {
      submitCopyToMeal(sourceDate, sourceMeal, sourcePosition, targetDate, targetMeal, "append");
      setCopyToMeal({ kind: "closed" });
      return;
    }
    setCopyToMeal({
      kind: "confirm-conflict",
      sourceDate,
      sourceMeal,
      sourcePosition,
      targetDate,
      targetMeal,
    });
  };

  const resolveCopyToMealConflict = (mode: CopyConflictMode) => {
    if (copyToMeal.kind !== "confirm-conflict") return;
    if (mode === "cancel") {
      setCopyToMeal({ kind: "closed" });
      return;
    }
    const { sourceDate, sourceMeal, sourcePosition, targetDate, targetMeal } = copyToMeal;
    submitCopyToMeal(sourceDate, sourceMeal, sourcePosition, targetDate, targetMeal, mode);
    setCopyToMeal({ kind: "closed" });
  };

  const handleEntryAction = (
    date: string,
    meal: string,
    action: EntryAction,
    entry: Entry,
    days?: number,
  ) => {
    switch (action) {
      case "replace":
        setPicker({ kind: "replace", date, meal, position: entry.position });
        break;
      case "remove":
        removeMut.mutate({ date, meal, position: entry.position });
        break;
      case "view-detail":
        // S03 レシピ詳細ドロワーは別 proposal で実装。現状はエラー表示に留める。
        setErrorKey("pages.meal_plan.errors.unknown");
        break;
      case "copy-days":
        if (days) startCopy(date, meal, days, entry.position);
        break;
      case "split-batch":
        splitMut.mutate({ date, meal, position: entry.position });
        break;
      case "merge-batch":
        void openMergeDialog(date, meal, entry.position, entry.recipeId);
        break;
      case "copy-to-meal":
        setCopyToMeal({
          kind: "picking",
          sourceDate: date,
          sourceMeal: meal,
          sourcePosition: entry.position,
        });
        break;
    }
  };

  const handleMealAction = (date: string, meal: string, action: MealAction, days?: number) => {
    switch (action) {
      case "clear":
        clearMut.mutate({ date, meal });
        break;
      case "copy-days":
        if (days) startCopy(date, meal, days, -1);
        break;
      case "reorder":
        // 上下ボタン UI は後続タスクで足す。現状は unknown 表示。
        setErrorKey("pages.meal_plan.errors.unknown");
        break;
    }
  };

  const weekdayFmt = new Intl.DateTimeFormat(i18n.language, { weekday: "short" });
  const dateFmt = new Intl.DateTimeFormat(i18n.language, {
    month: "numeric",
    day: "numeric",
  });

  const showModeDialog = pendingCopy !== null && pendingCopy.shareBatch === undefined;
  const showConflictDialog =
    pendingCopy !== null &&
    pendingCopy.shareBatch !== undefined &&
    pendingCopy.conflicts.length > 0;

  return (
    <div className="flex flex-col gap-2">
      {errorKey && (
        <p className="rounded-md border border-red-200 bg-red-50 p-2 text-sm text-red-700">
          {t(errorKey)}
        </p>
      )}
      <div
        className="flex flex-col gap-3"
        data-testid="period-calendar"
        data-period-weeks={periodWeeks}
      >
        {dateBlocks.map((blockDates) => {
          const blockKey = blockDates[0].toISOString();
          return (
            <div
              key={blockKey}
              className="grid gap-1 text-sm"
              style={{
                gridTemplateColumns: `6rem repeat(${columnsPerBlock}, minmax(0, 1fr))`,
              }}
            >
              <div />
              {blockDates.map((d) => {
                const isToday = isSameLocalDate(d, today);
                return (
                  <div
                    key={d.toISOString()}
                    className={`flex flex-col items-center rounded-md p-2 text-center ${
                      isToday ? "bg-blue-50 text-blue-900" : "text-neutral-700"
                    }`}
                    aria-current={isToday ? "date" : undefined}
                  >
                    <span className="text-xs text-neutral-500">{weekdayFmt.format(d)}</span>
                    <span className="text-base font-semibold">{dateFmt.format(d)}</span>
                    {isToday && (
                      <span className="text-[10px] uppercase tracking-wide text-blue-700">
                        {t("pages.meal_plan.today")}
                      </span>
                    )}
                  </div>
                );
              })}

              {meals.map((meal) => (
                <div key={`row-${blockKey}-${meal}`} className="contents">
                  <div className="flex items-center justify-end pr-2 text-xs font-medium text-neutral-500">
                    {t(mealLabelKey(meal))}
                  </div>
                  {blockDates.map((d) => {
                    const date = formatDate(d);
                    const key = `${date}::${meal}`;
                    const cellEntries = grouped.get(key) ?? [];
                    return (
                      <MealPlanCell
                        key={key}
                        date={date}
                        meal={meal}
                        entries={cellEntries}
                        preset={preset}
                        onAddFirst={() => setPicker({ kind: "add", date, meal })}
                        onAddMore={() => setPicker({ kind: "add", date, meal })}
                        onEntryAction={(a, e, days) => handleEntryAction(date, meal, a, e, days)}
                        onMealAction={(a, days) => handleMealAction(date, meal, a, days)}
                      />
                    );
                  })}
                </div>
              ))}
            </div>
          );
        })}
      </div>

      <p className="text-xs text-neutral-500">{t("pages.meal_plan.batch.legend.leftover")}</p>

      <RecipePickerDialog
        open={picker.kind !== "closed"}
        onOpenChange={(next) => {
          if (!next) setPicker({ kind: "closed" });
        }}
        onSelect={(r) => {
          if (picker.kind === "add") {
            assignMut.mutate({ date: picker.date, meal: picker.meal, recipeID: r.id });
          } else if (picker.kind === "replace") {
            replaceMut.mutate({
              date: picker.date,
              meal: picker.meal,
              position: picker.position,
              recipeID: r.id,
            });
          }
          setPicker({ kind: "closed" });
        }}
      />

      <CopyModeDialog
        open={showModeDialog}
        onOpenChange={(next) => {
          if (!next) setPendingCopy(null);
        }}
        onSelect={resolveMode}
      />

      <CopyConflictDialog
        open={showConflictDialog}
        conflictTargets={pendingCopy?.conflicts ?? []}
        onSelect={resolveConflict}
      />

      <BatchMergeDialog
        open={mergeDialog.kind === "open"}
        candidates={mergeDialog.kind === "open" ? mergeDialog.candidates : []}
        onOpenChange={(next) => {
          if (!next) setMergeDialog({ kind: "closed" });
        }}
        onSelect={(batchID) => {
          if (mergeDialog.kind === "open") {
            mergeMut.mutate({
              date: mergeDialog.date,
              meal: mergeDialog.meal,
              position: mergeDialog.position,
              targetBatchID: batchID,
            });
          }
          setMergeDialog({ kind: "closed" });
        }}
      />

      <CopyToMealDialog
        open={copyToMeal.kind === "picking"}
        sourceDate={copyToMeal.kind === "picking" ? copyToMeal.sourceDate : ""}
        sourceMeal={copyToMeal.kind === "picking" ? copyToMeal.sourceMeal : ""}
        dayRange={CROSS_MEAL_DAY_RANGE}
        onOpenChange={(next) => {
          if (!next) setCopyToMeal({ kind: "closed" });
        }}
        onSelect={handleCopyToMealPick}
      />

      <CopyConflictDialog
        open={copyToMeal.kind === "confirm-conflict"}
        conflictTargets={copyToMeal.kind === "confirm-conflict" ? [copyToMeal.targetDate] : []}
        onSelect={resolveCopyToMealConflict}
      />
    </div>
  );
}
