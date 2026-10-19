import { useTranslation } from "react-i18next";
import type { Entry } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { formatMealDate } from "../../lib/format-meal-date";
import { CellOverflowList } from "./cell-overflow-list";
import { DND_MIME, MealItemRow } from "./meal-item-row";
import { MealMenu } from "./meal-menu";
import { type CellSizePreset, mealLabelKey, minHeightClass } from "./mealplan-config";

export type EntryAction =
  | "toggle-cooked"
  | "replace"
  | "remove"
  | "view-detail"
  | "move-up"
  | "move-down"
  | "spread"
  | "split-batch"
  | "merge-batch";
export type MealAction = "clear" | "copy-days";

// id で操作する品を指す。date / meal / position は同じセルかの判定とドロップ位置の計算に使う (add-mealplan-entry-id design.md D7)。
export type DragSource = { id: string; date: string; meal: string; position: number };
export type DropTarget = { date: string; meal: string; position?: number };

interface Props {
  date: string;
  meal: string;
  entries: Entry[];
  preset: CellSizePreset;
  onAddFirst: () => void;
  onAddMore: () => void;
  onEntryAction: (action: EntryAction, entry: Entry) => void;
  onMealAction: (action: MealAction, days?: number) => void;
  isAnyDragging: boolean;
  dragSource: DragSource | null;
  onEntryDragStart: (source: DragSource) => void;
  onEntryDragEnd: () => void;
  onCellDrop: (target: DropTarget, sourceJson: string) => void;
}

// 1 セル = 1 (date, meal)。0 品なら中央に `＋`、1..N 品なら品行 (「作った」チェックボックス + レシピ名) +
// 右上 `＋` + 右下時間帯 `⋯`。品数超過は「他 N 件」で折り畳む。
// 「他 N 件」を押すと、そのセルの全品の一覧が開き、畳まれた品にも同じ操作が届く。
// バッチ (batch_id) が複数行に跨る品は枠色だけで示す (spec「作り置きの枠色による識別」)。
// セル高 / 文字サイズ / 表示件数上限は preset (期間 × 時間帯数) で切り替わる。
export function MealPlanCell({
  date,
  meal,
  entries,
  preset,
  onAddFirst,
  onAddMore,
  onEntryAction,
  onMealAction,
  isAnyDragging,
  dragSource,
  onEntryDragStart,
  onEntryDragEnd,
  onCellDrop,
}: Props) {
  const { t, i18n } = useTranslation();
  const limit = preset.visibleItems;
  const heightCls = minHeightClass(preset);
  const fontCls = preset.fontSizeClass;
  // ドラッグ中は全セルに枠強調を出し、drop 可能であることを示す (design.md D5)。
  const dropOutlineCls = isAnyDragging ? "outline outline-2 outline-blue-400" : "";

  const handleDragOver = (e: React.DragEvent<HTMLDivElement>) => {
    if (!isAnyDragging) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
  };

  const handleCellDrop = (e: React.DragEvent<HTMLDivElement>, position?: number) => {
    e.preventDefault();
    const raw = e.dataTransfer.getData(DND_MIME);
    if (!raw) return;
    onCellDrop({ date, meal, position }, raw);
  };

  if (entries.length === 0) {
    return (
      // biome-ignore lint/a11y/useSemanticElements: セルの drop 領域は <fieldset> にできない (レイアウト用の div)
      <div
        role="group"
        data-date={date}
        data-meal={meal}
        className={`${heightCls} rounded-md border border-dashed border-neutral-200 bg-white ${dropOutlineCls}`}
        aria-label={isAnyDragging ? t("pages.meal_plan.dnd.cell_drop_label") : undefined}
        onDragOver={handleDragOver}
        onDrop={(e) => handleCellDrop(e)}
      >
        <button
          type="button"
          className={`flex h-full ${heightCls} w-full items-center justify-center text-2xl text-neutral-500 hover:bg-neutral-50 hover:text-neutral-700`}
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
    // biome-ignore lint/a11y/useSemanticElements: セルの drop 領域は <fieldset> にできない (レイアウト用の div)
    <div
      role="group"
      data-date={date}
      data-meal={meal}
      className={`flex ${heightCls} flex-col gap-1 rounded-md border border-neutral-200 bg-white p-1 ${dropOutlineCls}`}
      aria-label={isAnyDragging ? t("pages.meal_plan.dnd.cell_drop_label") : undefined}
      onDragOver={handleDragOver}
      onDrop={(e) => handleCellDrop(e)}
    >
      <div className="flex justify-end">
        <button
          type="button"
          className="rounded p-1 text-sm leading-none text-neutral-500 hover:bg-neutral-100 hover:text-neutral-800"
          onClick={onAddMore}
          aria-label={t("pages.meal_plan.cell_actions.add_more")}
        >
          ＋
        </button>
      </div>
      {visible.map((entry) => {
        // batchMemberCount > 1 の作り置き行は個別移動を禁止する (design.md D1)。移動したい場合は
        // 「作り置きをやめる」→ 残った最初の日の行を D&D →「作り置きを広げる」の順で扱う。
        const isBatchMember = entry.batchMemberCount > 1;
        const isDragSource = dragSource !== null && dragSource.id === entry.id;
        return (
          <MealItemRow
            key={entry.id}
            entry={entry}
            cellItemCount={entries.length}
            fontClass={fontCls}
            draggable={!isBatchMember}
            isDragSource={isDragSource}
            onAction={onEntryAction}
            onDragStart={onEntryDragStart}
            onDragEnd={onEntryDragEnd}
            onDropOnRow={handleCellDrop}
          />
        );
      })}
      {overflow > 0 && (
        <CellOverflowList
          sortedEntries={sorted}
          overflow={overflow}
          label={t("pages.meal_plan.cell.overflow_list_label", {
            date: formatMealDate(date, i18n.language),
            meal: t(mealLabelKey(meal)),
          })}
          onEntryAction={onEntryAction}
        />
      )}
      <div className="mt-auto flex justify-end">
        <MealMenu
          onClear={() => onMealAction("clear")}
          onCopyDays={(days) => onMealAction("copy-days", days)}
        />
      </div>
    </div>
  );
}
