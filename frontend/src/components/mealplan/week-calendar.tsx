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
import { errorI18nKey } from "./error-messages";
import { type EntryAction, type MealAction, MealPlanCell } from "./meal-plan-cell";
import { MEAL_RENDER_ORDER, pickCellPreset } from "./mealplan-config";
import { RecipePickerDialog } from "./recipe-picker-dialog";
import { formatDate, isSameLocalDate, weekDates } from "./week-utils";

type Props = {
  weekStart: Date;
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

export function WeekCalendar({ weekStart, entries, visibleMeals }: Props) {
  const { t, i18n } = useTranslation();
  const qc = useQueryClient();
  const [picker, setPicker] = useState<PickerState>({ kind: "closed" });
  const [pendingCopy, setPendingCopy] = useState<PendingCopy | null>(null);
  const [mergeDialog, setMergeDialog] = useState<MergeDialogState>({ kind: "closed" });
  const [errorKey, setErrorKey] = useState<string | null>(null);

  const invalidate = () => qc.invalidateQueries({ queryKey: ["mealplan", "week"] });

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

  const dates = weekDates(weekStart);
  const today = new Date();
  const meals = MEAL_RENDER_ORDER.filter((s) => visibleMeals.has(s));
  // 期間切替 UI は未実装 (後続 `add-mealplan-period-switch` で state から取る)。
  // 現状は 1 週固定で決め打ち。preset は「期間 × 表示時間帯数」で解決する。
  const preset = pickCellPreset(1, meals.length);

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
      const candidates = (await MealPlanService.ListMealBatches(date, meal, recipeID)) ?? [];
      setMergeDialog({ kind: "open", date, meal, position, candidates });
      setErrorKey(null);
    } catch (err) {
      setErrorKey(errorI18nKey(err));
    }
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
        className="grid gap-1 text-sm"
        style={{ gridTemplateColumns: "6rem repeat(7, minmax(0, 1fr))" }}
      >
        <div />
        {dates.map((d) => {
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
          <div key={`row-${meal}`} className="contents">
            <div className="flex items-center justify-end pr-2 text-xs font-medium text-neutral-500">
              {t(mealLabelKey(meal))}
            </div>
            {dates.map((d) => {
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
    </div>
  );
}
