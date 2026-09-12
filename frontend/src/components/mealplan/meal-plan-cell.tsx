import { useTranslation } from "react-i18next";
import type { Entry } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { colorForBatch } from "./batch-color";
import { CellItemMenu } from "./cell-item-menu";
import { MealMenu } from "./meal-menu";
import { MEALPLAN_DEFAULTS } from "./mealplan-config";

export type EntryAction =
  | "replace"
  | "remove"
  | "view-detail"
  | "copy-days"
  | "split-batch"
  | "merge-batch";
export type MealAction = "clear" | "copy-days" | "reorder";

interface Props {
  date: string;
  meal: string;
  entries: Entry[];
  onAddFirst: () => void;
  onAddMore: () => void;
  onEntryAction: (action: EntryAction, entry: Entry, days?: number) => void;
  onMealAction: (action: MealAction, days?: number) => void;
}

// 1 セル = 1 (date, meal)。0 品なら中央に `＋`、1..N 品ならレシピ名列 + 品ごと `⋯` +
// 右上 `＋` + 右下時間帯 `⋯`。品数超過は「他 N 件」で折り畳む。
// バッチ (batch_id) が複数行に跨る品には枠色 + 🔗 アイコンを描画する (spec の視覚識別要件)。
export function MealPlanCell({
  entries,
  onAddFirst,
  onAddMore,
  onEntryAction,
  onMealAction,
}: Props) {
  const { t } = useTranslation();
  const limit = MEALPLAN_DEFAULTS.mealPlanCellVisibleItems;

  if (entries.length === 0) {
    return (
      <div className="min-h-16 rounded-md border border-dashed border-neutral-200 bg-white">
        <button
          type="button"
          className="flex h-full min-h-16 w-full items-center justify-center text-sm text-neutral-400 hover:bg-neutral-50 hover:text-neutral-700"
          onClick={onAddFirst}
          aria-label={t("pages.meal_plan.cell_actions.add_first")}
        >
          ＋
        </button>
      </div>
    );
  }

  const sorted = [...entries].sort((a, b) => a.position - b.position);
  const visible = sorted.slice(0, limit);
  const overflow = sorted.length - visible.length;

  return (
    <div className="relative flex min-h-16 flex-col gap-1 rounded-md border border-neutral-200 bg-white p-2">
      <button
        type="button"
        className="absolute right-1 top-1 text-xs text-neutral-400 hover:text-neutral-700"
        onClick={onAddMore}
        aria-label={t("pages.meal_plan.cell_actions.add_more")}
      >
        ＋
      </button>
      {visible.map((entry) => {
        const color = colorForBatch(entry.batchId, entry.batchMemberCount);
        return (
          <div
            key={entry.position}
            className={`flex items-center justify-between gap-1 rounded-sm border px-1 py-0.5 text-sm text-neutral-800 ${
              color ? `${color}` : "border-transparent"
            }`}
            data-batch-id={entry.batchId}
            data-batch-member-count={entry.batchMemberCount}
          >
            <span className="flex min-w-0 items-center gap-1">
              {color && (
                <span aria-hidden="true" className="text-xs">
                  🔗
                </span>
              )}
              <span className="truncate">{entry.recipeName}</span>
            </span>
            <CellItemMenu
              batchMemberCount={entry.batchMemberCount}
              onReplace={() => onEntryAction("replace", entry)}
              onRemove={() => onEntryAction("remove", entry)}
              onViewDetail={() => onEntryAction("view-detail", entry)}
              onCopyDays={(days) => onEntryAction("copy-days", entry, days)}
              onSplitBatch={() => onEntryAction("split-batch", entry)}
              onMergeBatch={() => onEntryAction("merge-batch", entry)}
            />
          </div>
        );
      })}
      {overflow > 0 && (
        <div className="text-xs text-neutral-500">
          {t("pages.meal_plan.cell.more_items", { count: overflow })}
        </div>
      )}
      <div className="mt-auto flex justify-end">
        <MealMenu
          onClear={() => onMealAction("clear")}
          onCopyDays={(days) => onMealAction("copy-days", days)}
          onReorder={() => onMealAction("reorder")}
        />
      </div>
    </div>
  );
}
