import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type {
  BatchSummary,
  Entry,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { Slot } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { MealPlanService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { BatchMergeDialog } from "./batch-merge-dialog";
import { CopyConflictDialog, type CopyConflictMode } from "./copy-conflict-dialog";
import { CopyModeDialog } from "./copy-mode-dialog";
import { previewCopy } from "./copy-planner";
import { errorI18nKey } from "./error-messages";
import { type EntryAction, MealPlanCell, type SlotAction } from "./meal-plan-cell";
import { SLOT_RENDER_ORDER } from "./mealplan-config";
import { RecipePickerDialog } from "./recipe-picker-dialog";
import { formatDate, isSameLocalDate, weekDates } from "./week-utils";

type Props = {
  weekStart: Date;
  entries: Entry[];
  visibleSlots: Set<string>;
};

type PickerState =
  | { kind: "closed" }
  | { kind: "add"; date: string; slot: string }
  | { kind: "replace"; date: string; slot: string; position: number };

type PendingCopy = {
  date: string;
  slot: string;
  days: number;
  position: number; // -1 = slot 全体
  targets: string[];
  conflicts: string[];
  shareBatch?: boolean;
};

type MergeDialogState =
  | { kind: "closed" }
  | { kind: "open"; date: string; slot: string; position: number; candidates: BatchSummary[] };

function slotLabelKey(slot: string): string {
  switch (slot) {
    case Slot.SlotBreakfast:
      return "pages.meal_plan.slot.breakfast";
    case Slot.SlotLunch:
      return "pages.meal_plan.slot.lunch";
    default:
      return "pages.meal_plan.slot.dinner";
  }
}

export function WeekCalendar({ weekStart, entries, visibleSlots }: Props) {
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
    mutationFn: (v: { date: string; slot: string; recipeID: string }) =>
      withHandlers(MealPlanService.AssignRecipe(v.date, v.slot, v.recipeID)),
  });
  const replaceMut = useMutation({
    mutationFn: (v: { date: string; slot: string; position: number; recipeID: string }) =>
      withHandlers(MealPlanService.ReplaceEntry(v.date, v.slot, v.position, v.recipeID)),
  });
  const removeMut = useMutation({
    mutationFn: (v: { date: string; slot: string; position: number }) =>
      withHandlers(MealPlanService.RemoveEntry(v.date, v.slot, v.position)),
  });
  const clearMut = useMutation({
    mutationFn: (v: { date: string; slot: string }) =>
      withHandlers(MealPlanService.ClearSlot(v.date, v.slot)),
  });
  const copyMut = useMutation({
    mutationFn: (v: {
      date: string;
      slot: string;
      days: number;
      position: number;
      mode: string;
      shareBatch: boolean;
    }) =>
      withHandlers(
        MealPlanService.CopyLeftover(v.date, v.slot, v.days, v.position, v.mode, v.shareBatch),
      ),
  });
  const splitMut = useMutation({
    mutationFn: (v: { date: string; slot: string; position: number }) =>
      withHandlers(MealPlanService.SplitBatch(v.date, v.slot, v.position)),
  });
  const mergeMut = useMutation({
    mutationFn: (v: { date: string; slot: string; position: number; targetBatchID: string }) =>
      withHandlers(MealPlanService.MergeBatch(v.date, v.slot, v.position, v.targetBatchID)),
  });

  const dates = weekDates(weekStart);
  const today = new Date();
  const slots = SLOT_RENDER_ORDER.filter((s) => visibleSlots.has(s));

  const grouped = new Map<string, Entry[]>();
  for (const e of entries) {
    const key = `${e.date}::${e.slot}`;
    const bucket = grouped.get(key);
    if (bucket) bucket.push(e);
    else grouped.set(key, [e]);
  }

  const startCopy = (date: string, slot: string, days: number, position: number) => {
    const { targets, conflicts } = previewCopy(date, slot, days, entries);
    if (targets.length === 0) return;
    setPendingCopy({ date, slot, days, position, targets, conflicts });
  };

  const resolveMode = (shareBatch: boolean) => {
    if (!pendingCopy) return;
    if (pendingCopy.conflicts.length === 0) {
      copyMut.mutate({
        date: pendingCopy.date,
        slot: pendingCopy.slot,
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
      slot: pendingCopy.slot,
      days: pendingCopy.days,
      position: pendingCopy.position,
      mode,
      shareBatch: pendingCopy.shareBatch ?? true,
    });
    setPendingCopy(null);
  };

  const openMergeDialog = async (
    date: string,
    slot: string,
    position: number,
    recipeID: string,
  ) => {
    try {
      const candidates = (await MealPlanService.ListSlotBatches(date, slot, recipeID)) ?? [];
      setMergeDialog({ kind: "open", date, slot, position, candidates });
      setErrorKey(null);
    } catch (err) {
      setErrorKey(errorI18nKey(err));
    }
  };

  const handleEntryAction = (
    date: string,
    slot: string,
    action: EntryAction,
    entry: Entry,
    days?: number,
  ) => {
    switch (action) {
      case "replace":
        setPicker({ kind: "replace", date, slot, position: entry.position });
        break;
      case "remove":
        removeMut.mutate({ date, slot, position: entry.position });
        break;
      case "view-detail":
        // S03 レシピ詳細ドロワーは別 proposal で実装。現状はエラー表示に留める。
        setErrorKey("pages.meal_plan.errors.unknown");
        break;
      case "copy-days":
        if (days) startCopy(date, slot, days, entry.position);
        break;
      case "split-batch":
        splitMut.mutate({ date, slot, position: entry.position });
        break;
      case "merge-batch":
        void openMergeDialog(date, slot, entry.position, entry.recipeId);
        break;
    }
  };

  const handleSlotAction = (date: string, slot: string, action: SlotAction, days?: number) => {
    switch (action) {
      case "clear":
        clearMut.mutate({ date, slot });
        break;
      case "copy-days":
        if (days) startCopy(date, slot, days, -1);
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

        {slots.map((slot) => (
          <div key={`row-${slot}`} className="contents">
            <div className="flex items-center justify-end pr-2 text-xs font-medium text-neutral-500">
              {t(slotLabelKey(slot))}
            </div>
            {dates.map((d) => {
              const date = formatDate(d);
              const key = `${date}::${slot}`;
              const cellEntries = grouped.get(key) ?? [];
              return (
                <MealPlanCell
                  key={key}
                  date={date}
                  slot={slot}
                  entries={cellEntries}
                  onAddFirst={() => setPicker({ kind: "add", date, slot })}
                  onAddMore={() => setPicker({ kind: "add", date, slot })}
                  onEntryAction={(a, e, days) => handleEntryAction(date, slot, a, e, days)}
                  onSlotAction={(a, days) => handleSlotAction(date, slot, a, days)}
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
            assignMut.mutate({ date: picker.date, slot: picker.slot, recipeID: r.id });
          } else if (picker.kind === "replace") {
            replaceMut.mutate({
              date: picker.date,
              slot: picker.slot,
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
              slot: mergeDialog.slot,
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
