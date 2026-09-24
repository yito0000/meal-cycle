import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type {
  BatchSummary,
  Entry,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { Meal } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { MealPlanService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { planSpread, type SpreadSelection } from "../../lib/spread-batch-selection";
import { RecipeDetailDrawer } from "../recipe/recipe-detail-drawer";
import { BatchMergeDialog } from "./batch-merge-dialog";
import { CopyConflictDialog, type CopyConflictMode } from "./copy-conflict-dialog";
import { CopyModeDialog } from "./copy-mode-dialog";
import { previewCopy } from "./copy-planner";
import { errorI18nKey } from "./error-messages";
import {
  type DragSource,
  type DropTarget,
  type EntryAction,
  type MealAction,
  MealPlanCell,
} from "./meal-plan-cell";
import { MEAL_RENDER_ORDER, MEALPLAN_DEFAULTS, pickCellPreset } from "./mealplan-config";
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
  | { kind: "replace"; date: string; meal: string; position: number };

type PendingCopy = {
  date: string;
  meal: string;
  days: number;
  position: number;
  targets: string[];
  conflicts: string[];
  shareBatch?: boolean;
};

type MergeDialogState =
  | { kind: "closed" }
  | { kind: "open"; date: string; meal: string; position: number; candidates: BatchSummary[] };

type SpreadState =
  | { kind: "closed" }
  | {
      kind: "open";
      sourceDate: string;
      sourceMeal: string;
      sourcePosition: number;
      sourceRecipeName: string;
    };

// 統合ダイアログの対象日は起点日 + 未来 (max(LeftoverCopyPresets) - 1) 日 = 4 行 (実装時点)。
// 上限が max(presets)-1 なのは、CopyLeftover の days が LeftoverCopyPresets の要素のみ受け付ける
// ため (最大オフセット = max(presets) - 1 = 3)。過去方向は「作り置きは翌日以降に伸ばす」意図に
// 反するため対象外。expandStep=0 は「展開ボタンを出さない」意味 (backend 側 presets を拡張する
// 別 proposal で対応する予定)。
const SPREAD_INITIAL_FUTURE_DAYS = Math.max(...MEALPLAN_DEFAULTS.leftoverCopyPresets) - 1;
const SPREAD_EXPAND_STEP = 0;
const SPREAD_PRESET_DAYS = Math.min(...MEALPLAN_DEFAULTS.leftoverCopyPresets);

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

// 「連続 N 日」の N (起点日を含む合計日数) を、CopyLeftover の呼び出し形式に合わせて算出する。
// spread 選択の同 meal 対象日群は起点日 +1..+K の連続 K 日ではなく、任意のとびとびの可能性がある。
// CopyLeftover は「起点日を含む合計 N 日」の連続日を仮定するので、選択日群の最大オフセット + 1 を N に採る。
// 選択されなかった中間日は今回追加されないため、UI 側でとびとび選択されても「起点 +max オフセットまで
// 連続に追加された」ことになる (中間空セルは自動的に埋まる)。ここは統合ダイアログの D1 リスクを踏まえた
// 意図的挙動: 「連続 N 日」相当のセマンティクスに揃える (design.md D2)。
function computeCopyDays(sourceDate: string, sameMealDates: string[]): number {
  let maxOffset = 0;
  for (const d of sameMealDates) {
    const [sy, sm, sd] = sourceDate.split("-").map(Number);
    const [dy, dm, dd] = d.split("-").map(Number);
    const gap = Math.round(
      (new Date(dy, dm - 1, dd).getTime() - new Date(sy, sm - 1, sd).getTime()) / 86_400_000,
    );
    if (gap > maxOffset) maxOffset = gap;
  }
  return maxOffset + 1;
}

export function WeekCalendar({ periodStart, periodWeeks, entries, visibleMeals }: Props) {
  const { t, i18n } = useTranslation();
  const qc = useQueryClient();
  const [picker, setPicker] = useState<PickerState>({ kind: "closed" });
  const [pendingCopy, setPendingCopy] = useState<PendingCopy | null>(null);
  const [mergeDialog, setMergeDialog] = useState<MergeDialogState>({ kind: "closed" });
  const [spreadState, setSpreadState] = useState<SpreadState>({ kind: "closed" });
  const [errorState, setErrorState] = useState<{
    key: string;
    params?: Record<string, unknown>;
  } | null>(null);
  const [dragSource, setDragSource] = useState<DragSource | null>(null);
  // S01 から開いた S03 は閉じたら S01 に留まるだけなので、URL には載せない (add-recipe-details D2 / D8)。
  const [detailRecipeId, setDetailRecipeId] = useState<string | null>(null);
  const navigate = useNavigate();

  const setErrorKey = (key: string | null) => {
    if (key === null) setErrorState(null);
    else setErrorState({ key });
  };

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
  const reorderMut = useMutation({
    mutationFn: (v: { date: string; meal: string; orderedRecipeIDs: string[] }) =>
      withHandlers(MealPlanService.ReorderMeal(v.date, v.meal, v.orderedRecipeIDs)),
  });
  const moveMut = useMutation({
    mutationFn: (v: {
      sourceDate: string;
      sourceMeal: string;
      sourcePosition: number;
      targetDate: string;
      targetMeal: string;
    }) =>
      withHandlers(
        MealPlanService.MoveEntry(
          v.sourceDate,
          v.sourceMeal,
          v.sourcePosition,
          v.targetDate,
          v.targetMeal,
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

  // 統合ダイアログの OK 押下: 選択セル群を「同 meal (CopyLeftover)」と「別 meal (CopyToOtherMeal)」に
  // 分けて逐次反映する。SQLite への同時書き込みは modernc.org ドライバの busy_timeout に頼れば
  // 通ることが多いが、Wails IPC 経由で並列に投げると片方が確定前に他方の invalidate と競って
  // 起点行が見えなくなるケースがある。今は件数が少ない (最大 3 * 4 = 12) ので逐次で十分。
  const submitSpread = async (selection: SpreadSelection) => {
    if (spreadState.kind !== "open") return;
    const source = {
      date: spreadState.sourceDate,
      meal: spreadState.sourceMeal,
      position: spreadState.sourcePosition,
    };
    const { sameMealDates, crossMeal } = planSpread(source, selection);
    setSpreadState({ kind: "closed" });

    const conflictMode = selection.conflictMode;
    const shareBatch = selection.shareBatch;
    const tasks: (() => Promise<void>)[] = [];
    if (sameMealDates.length > 0) {
      const days = computeCopyDays(source.date, sameMealDates);
      tasks.push(async () => {
        await MealPlanService.CopyLeftover(
          source.date,
          source.meal,
          days,
          source.position,
          conflictMode,
          shareBatch,
        );
      });
    }
    for (const target of crossMeal) {
      tasks.push(async () => {
        await MealPlanService.CopyToOtherMeal(
          source.date,
          source.meal,
          source.position,
          target.date,
          target.meal,
          conflictMode,
        );
      });
    }

    let failed = 0;
    let firstErrKey: string | null = null;
    for (const run of tasks) {
      try {
        await run();
      } catch (err) {
        failed++;
        if (firstErrKey === null) firstErrKey = errorI18nKey(err);
        // 統合ダイアログの失敗は分類が難しい (source/target 組み合わせが混ざる) ので、
        // devtools console に生 error を残して個別に追えるようにする。
        console.error("spread submit: task failed", err);
      }
    }
    invalidate();
    if (failed > 0) {
      // 全部同じ理由で失敗した場合はその key を、混在するときは partial_failure に集約する。
      setErrorState(
        failed === tasks.length && firstErrKey !== null
          ? { key: firstErrKey }
          : { key: "pages.meal_plan.spread.partial_failure", params: { count: failed } },
      );
    } else {
      setErrorState(null);
    }
  };

  const handleEntryAction = (date: string, meal: string, action: EntryAction, entry: Entry) => {
    switch (action) {
      case "replace":
        setPicker({ kind: "replace", date, meal, position: entry.position });
        break;
      case "remove":
        removeMut.mutate({ date, meal, position: entry.position });
        break;
      case "view-detail":
        setDetailRecipeId(entry.recipeId);
        break;
      case "spread":
        setSpreadState({
          kind: "open",
          sourceDate: date,
          sourceMeal: meal,
          sourcePosition: entry.position,
          sourceRecipeName: entry.recipeName,
        });
        break;
      case "split-batch":
        splitMut.mutate({ date, meal, position: entry.position });
        break;
      case "merge-batch":
        void openMergeDialog(date, meal, entry.position, entry.recipeId);
        break;
    }
  };

  // 同一セル内 D&D は既存 ReorderMeal に流す (design.md D2)。drop 対象 entry の位置に source を差し込み、
  // 他の entry は相対順を維持して詰め直す。drop 対象 entry が無い (セル余白へのドロップ) は no-op。
  const reorderWithinCell = (
    date: string,
    meal: string,
    sourcePosition: number,
    targetPosition: number,
  ) => {
    if (sourcePosition === targetPosition) return;
    const cellEntries = entries
      .filter((e) => e.date === date && e.meal === meal)
      .sort((a, b) => a.position - b.position);
    if (cellEntries.length === 0) return;
    const withoutSource = cellEntries.filter((e) => e.position !== sourcePosition);
    const source = cellEntries.find((e) => e.position === sourcePosition);
    if (!source) return;
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
      orderedRecipeIDs: reordered.map((e) => e.recipeId),
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
      reorderWithinCell(source.date, source.meal, source.position, target.position);
      return;
    }
    moveMut.mutate({
      sourceDate: source.date,
      sourceMeal: source.meal,
      sourcePosition: source.position,
      targetDate: target.date,
      targetMeal: target.meal,
    });
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
      {errorState && (
        <p className="rounded-md border border-red-200 bg-red-50 p-2 text-sm text-red-700">
          {t(errorState.key, errorState.params)}
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

      {spreadState.kind === "open" && (
        <SpreadBatchDialog
          open
          source={{
            date: spreadState.sourceDate,
            meal: spreadState.sourceMeal,
            position: spreadState.sourcePosition,
          }}
          sourceRecipeName={spreadState.sourceRecipeName}
          existingEntries={entries}
          initialFutureDays={SPREAD_INITIAL_FUTURE_DAYS}
          expandStepDays={SPREAD_EXPAND_STEP}
          presetDays={SPREAD_PRESET_DAYS}
          onOpenChange={(next) => {
            if (!next) setSpreadState({ kind: "closed" });
          }}
          onConfirm={(selection) => {
            void submitSpread(selection);
          }}
        />
      )}
      <RecipeDetailDrawer
        recipeId={detailRecipeId}
        onClose={() => setDetailRecipeId(null)}
        onEdit={(id) => navigate({ to: "/recipes/$recipeId/edit", params: { recipeId: id } })}
      />
    </div>
  );
}
