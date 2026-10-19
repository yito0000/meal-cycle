import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import type {
  BatchSummary,
  Entry,
  Meal,
  RemovedEntry,
  SpreadResult,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { MealPlanService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";
import type { SpreadSelection, SpreadSource } from "../../lib/spread-batch-selection";
import { UNDO_TOAST_MS } from "../../lib/undo-toast";
import { useSettingsValues } from "../settings/settings-gate";
import { BatchMergeDialog } from "./batch-merge-dialog";
import { CopyConflictDialog, type CopyConflictMode } from "./copy-conflict-dialog";
import { CopyModeDialog } from "./copy-mode-dialog";
import { previewCopy } from "./copy-planner";
import { DissolveBatchDialog } from "./dissolve-batch-dialog";
import { errorI18nKey } from "./error-messages";
import {
  type DragSource,
  type DropTarget,
  type EntryAction,
  type MealAction,
  MealPlanCell,
} from "./meal-plan-cell";
import { MEAL_RENDER_ORDER, mealLabelKey, pickCellPreset } from "./mealplan-config";
import type { PeriodWeeks } from "./period-nav";
import { RecipePickerDialog } from "./recipe-picker-dialog";
import { SpreadBatchDialog } from "./spread-batch-dialog";
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
  | { kind: "replace"; entryId: string };

type PendingCopy = {
  date: string;
  meal: string;
  days: number;
  targets: string[];
  conflicts: string[];
  shareBatch?: boolean;
};

type MergeDialogState =
  | { kind: "closed" }
  | { kind: "open"; entryId: string; candidates: BatchSummary[] };

type DissolveState = { kind: "closed" } | { kind: "open"; entryId: string; removedCount: number };

// 1 品に対する操作は行の ID で指す。並び順は削除や並べ替えで振り直され、取り直す前の一覧の並び順は
// 別の品を指しうるため (add-mealplan-entry-id)。
type SpreadState =
  | { kind: "closed" }
  | {
      kind: "open";
      source: SpreadSource;
      sourceRecipeName: string;
      sourceRecipeId: string;
      sourceBatchId: string;
    };

export function WeekCalendar({ periodStart, periodWeeks, entries, visibleMeals }: Props) {
  const { t, i18n } = useTranslation();
  const { leftoverCopyPresets } = useSettingsValues();
  const spreadPresetDays = Math.min(...leftoverCopyPresets);
  const qc = useQueryClient();
  const [picker, setPicker] = useState<PickerState>({ kind: "closed" });
  const [pendingCopy, setPendingCopy] = useState<PendingCopy | null>(null);
  const [mergeDialog, setMergeDialog] = useState<MergeDialogState>({ kind: "closed" });
  const [spreadState, setSpreadState] = useState<SpreadState>({ kind: "closed" });
  const [dissolveState, setDissolveState] = useState<DissolveState>({ kind: "closed" });
  const [dragSource, setDragSource] = useState<DragSource | null>(null);
  const navigate = useNavigate();

  // 期間切替と前後ナビの再取得は queryKey ["mealplan", "period", …] を共通 prefix ["mealplan"] で
  // 全部落とすように統一する (design.md D4)。
  const invalidate = () => qc.invalidateQueries({ queryKey: ["mealplan"] });

  const withHandlers = <T,>(promise: Promise<T>) =>
    promise
      .then((result) => {
        invalidate();
        return result;
      })
      .catch((err) => {
        toast.error(t(errorI18nKey(err)));
        throw err;
      });

  const showUndoToast = (message: string, removed: RemovedEntry[]) => {
    toast(message, {
      duration: UNDO_TOAST_MS,
      action: {
        label: t("pages.meal_plan.toast.undo"),
        onClick: () => restoreMut.mutate(removed),
      },
    });
  };

  const assignMut = useMutation({
    mutationFn: (v: { date: string; meal: string; recipeID: string }) =>
      withHandlers(MealPlanService.AssignRecipe(v.date, v.meal, v.recipeID)),
  });
  const replaceMut = useMutation({
    mutationFn: (v: { entryId: string; recipeID: string }) =>
      withHandlers(MealPlanService.ReplaceEntry(v.entryId, v.recipeID)),
  });
  // 通知は mutate ごとではなく useMutation 側で出す。mutate に渡したコールバックは最後の呼び出しの分しか
  // 呼ばれず、続けて 2 品を削除すると 1 品目の「元に戻す」が出なくなるため。
  const removeMut = useMutation({
    mutationFn: (v: { entryId: string; recipeName: string }) =>
      withHandlers(MealPlanService.RemoveEntry(v.entryId)),
    onSuccess: (removed, v) =>
      showUndoToast(t("pages.meal_plan.toast.removed", { name: v.recipeName }), [removed]),
  });
  const clearMut = useMutation({
    mutationFn: (v: { date: string; meal: string }) =>
      withHandlers(MealPlanService.ClearMeal(v.date, v.meal)),
    onSuccess: (removed) => {
      if (!removed || removed.length === 0) return;
      showUndoToast(t("pages.meal_plan.toast.cleared", { count: removed.length }), removed);
    },
  });
  // S05 と同じく「戻せなかった」を見出しにし、理由は説明に添える。
  const restoreMut = useMutation({
    mutationFn: (entries: RemovedEntry[]) => MealPlanService.RestoreEntries(entries),
    onSuccess: () => invalidate(),
    onError: (err) =>
      toast.error(t("pages.meal_plan.toast.restore_failed"), {
        description: t(errorI18nKey(err)),
      }),
  });
  const copyMut = useMutation({
    mutationFn: (v: {
      date: string;
      meal: string;
      days: number;
      mode: string;
      shareBatch: boolean;
    }) => withHandlers(MealPlanService.CopyLeftover(v.date, v.meal, v.days, v.mode, v.shareBatch)),
  });
  const spreadMut = useMutation({
    mutationFn: (v: { entryId: string; selection: SpreadSelection }) =>
      withHandlers(
        MealPlanService.SpreadEntry(
          v.entryId,
          v.selection.targets.map((c) => ({ date: c.date, meal: c.meal as Meal })),
          v.selection.conflictMode,
          v.selection.shareBatch,
        ),
      ),
    onSuccess: (result) => {
      const replaced = result.removed?.length ?? 0;
      const cells = result.addedEntryIds?.length ?? 0;
      const message =
        replaced > 0
          ? t("pages.meal_plan.toast.spread_overwritten", { cells, replaced })
          : t("pages.meal_plan.toast.spread", { cells });
      toast(message, {
        duration: UNDO_TOAST_MS,
        action: {
          label: t("pages.meal_plan.toast.undo"),
          onClick: () => undoSpreadMut.mutate(result),
        },
      });
    },
  });
  const undoSpreadMut = useMutation({
    mutationFn: (result: SpreadResult) => MealPlanService.UndoSpread(result),
    onSuccess: () => invalidate(),
    onError: (err) =>
      toast.error(t("pages.meal_plan.toast.restore_failed"), {
        description: t(errorI18nKey(err)),
      }),
  });
  const cookedMut = useMutation({
    mutationFn: (v: { entryId: string; cooked: boolean }) =>
      withHandlers(MealPlanService.SetCooked(v.entryId, v.cooked)),
  });
  const splitMut = useMutation({
    mutationFn: (v: { entryId: string }) => withHandlers(MealPlanService.SplitBatch(v.entryId)),
  });
  const mergeMut = useMutation({
    mutationFn: (v: { entryId: string; targetBatchID: string }) =>
      withHandlers(MealPlanService.MergeBatch(v.entryId, v.targetBatchID)),
  });
  const reorderMut = useMutation({
    mutationFn: (v: { date: string; meal: string; orderedEntryIDs: string[] }) =>
      withHandlers(MealPlanService.ReorderMeal(v.date, v.meal, v.orderedEntryIDs)),
  });
  const moveMut = useMutation({
    mutationFn: (v: { entryId: string; targetDate: string; targetMeal: string }) =>
      withHandlers(MealPlanService.MoveEntry(v.entryId, v.targetDate, v.targetMeal)),
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

  const startCopy = (date: string, meal: string, days: number) => {
    const { targets, conflicts } = previewCopy(date, meal, days, entries);
    if (targets.length === 0) return;
    setPendingCopy({ date, meal, days, targets, conflicts });
  };

  const resolveMode = (shareBatch: boolean) => {
    if (!pendingCopy) return;
    if (pendingCopy.conflicts.length === 0) {
      copyMut.mutate({
        date: pendingCopy.date,
        meal: pendingCopy.meal,
        days: pendingCopy.days,
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
      mode,
      shareBatch: pendingCopy.shareBatch ?? true,
    });
    setPendingCopy(null);
  };

  const openMergeDialog = async (entryId: string) => {
    try {
      const candidates = (await MealPlanService.ListMergeCandidates(entryId)) ?? [];
      setMergeDialog({ kind: "open", entryId, candidates });
    } catch (err) {
      toast.error(t(errorI18nKey(err)));
    }
  };

  const submitSpread = (selection: SpreadSelection) => {
    if (spreadState.kind !== "open") return;
    setSpreadState({ kind: "closed" });
    spreadMut.mutate({ entryId: spreadState.source.id, selection });
  };

  const handleEntryAction = (date: string, meal: string, action: EntryAction, entry: Entry) => {
    switch (action) {
      case "toggle-cooked":
        cookedMut.mutate({ entryId: entry.id, cooked: !entry.cooked });
        break;
      case "replace":
        setPicker({ kind: "replace", entryId: entry.id });
        break;
      case "remove":
        removeMut.mutate({ entryId: entry.id, recipeName: entry.recipeName });
        break;
      case "view-detail":
        navigate({
          to: "/recipes/$recipeId",
          params: { recipeId: entry.recipeId },
          search: { back: { to: "mealplan", start: formatDate(periodStart) } },
        });
        break;
      case "move-up":
        reorderWithinCell(date, meal, entry.id, entry.position - 1);
        break;
      case "move-down": {
        // reorderWithinCell は source を target の前に差し込むので、1 つ下の品を上に動かす形で渡す。
        const below = entries.find(
          (e) => e.date === date && e.meal === meal && e.position === entry.position + 1,
        );
        if (below) reorderWithinCell(date, meal, below.id, entry.position);
        break;
      }
      case "spread":
        setSpreadState({
          kind: "open",
          source: { id: entry.id, date, meal },
          sourceRecipeName: entry.recipeName,
          sourceRecipeId: entry.recipeId,
          sourceBatchId: entry.batchId,
        });
        break;
      case "split-batch":
        setDissolveState({
          kind: "open",
          entryId: entry.id,
          removedCount: entry.batchMemberCount - 1,
        });
        break;
      case "merge-batch":
        void openMergeDialog(entry.id);
        break;
    }
  };

  // 同一セル内 D&D は既存 ReorderMeal に流す (design.md D2)。drop 対象 entry の位置に source を差し込み、
  // 他の entry は相対順を維持して詰め直す。drop 対象 entry が無い (セル余白へのドロップ) は no-op。
  // 並びは行の ID で渡すので、同じレシピの品が 2 つあっても区別して並べ替えられる。
  const reorderWithinCell = (
    date: string,
    meal: string,
    sourceId: string,
    targetPosition: number,
  ) => {
    const cellEntries = entries
      .filter((e) => e.date === date && e.meal === meal)
      .sort((a, b) => a.position - b.position);
    const source = cellEntries.find((e) => e.id === sourceId);
    if (!source || source.position === targetPosition) return;
    const withoutSource = cellEntries.filter((e) => e.id !== sourceId);
    // targetPosition の元の index に source を挿入する。target が source より後ろだった場合、
    // source を先に抜くことで index が 1 つ前にずれる点はそのまま利用する。
    const targetIdx = withoutSource.findIndex((e) => e.position === targetPosition);
    const insertAt = targetIdx < 0 ? withoutSource.length : targetIdx;
    const reordered = [
      ...withoutSource.slice(0, insertAt),
      source,
      ...withoutSource.slice(insertAt),
    ];
    reorderMut.mutate({
      date,
      meal,
      orderedEntryIDs: reordered.map((e) => e.id),
    });
  };

  const handleDropEntry = (target: DropTarget, sourceJson: string) => {
    let source: DragSource;
    try {
      source = JSON.parse(sourceJson) as DragSource;
    } catch {
      return;
    }
    setDragSource(null);
    if (source.date === target.date && source.meal === target.meal) {
      if (target.position === undefined) return;
      reorderWithinCell(source.date, source.meal, source.id, target.position);
      return;
    }
    moveMut.mutate({ entryId: source.id, targetDate: target.date, targetMeal: target.meal });
  };

  const handleMealAction = (date: string, meal: string, action: MealAction, days?: number) => {
    switch (action) {
      case "clear":
        clearMut.mutate({ date, meal });
        break;
      case "copy-days":
        if (days) startCopy(date, meal, days);
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
                gridTemplateColumns: `2.5rem repeat(${columnsPerBlock}, minmax(0, 1fr))`,
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
                        onEntryAction={(a, e) => handleEntryAction(date, meal, a, e)}
                        onMealAction={(a, days) => handleMealAction(date, meal, a, days)}
                        isAnyDragging={dragSource !== null}
                        dragSource={dragSource}
                        onEntryDragStart={(source) => setDragSource(source)}
                        onEntryDragEnd={() => setDragSource(null)}
                        onCellDrop={handleDropEntry}
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
            replaceMut.mutate({ entryId: picker.entryId, recipeID: r.id });
          }
          setPicker({ kind: "closed" });
        }}
      />

      {/* 時間帯全体の連続 N 日コピーは統合ダイアログの対象外 (proposal Non-goal)。CopyModeDialog は
          そちらの経路で使い続ける (design.md 追記の判断)。 */}
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
            mergeMut.mutate({ entryId: mergeDialog.entryId, targetBatchID: batchID });
          }
          setMergeDialog({ kind: "closed" });
        }}
      />

      <DissolveBatchDialog
        open={dissolveState.kind === "open"}
        removedCount={dissolveState.kind === "open" ? dissolveState.removedCount : 0}
        onCancel={() => setDissolveState({ kind: "closed" })}
        onConfirm={() => {
          if (dissolveState.kind === "open") {
            splitMut.mutate({ entryId: dissolveState.entryId });
          }
          setDissolveState({ kind: "closed" });
        }}
      />

      {spreadState.kind === "open" && (
        <SpreadBatchDialog
          open
          source={spreadState.source}
          sourceRecipeName={spreadState.sourceRecipeName}
          sourceRecipeId={spreadState.sourceRecipeId}
          sourceBatchId={spreadState.sourceBatchId}
          presetDays={spreadPresetDays}
          onOpenChange={(next) => {
            if (!next) setSpreadState({ kind: "closed" });
          }}
          onConfirm={(selection) => {
            submitSpread(selection);
          }}
        />
      )}
    </div>
  );
}
