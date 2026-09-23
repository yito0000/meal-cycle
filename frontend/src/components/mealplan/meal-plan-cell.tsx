import { useTranslation } from "react-i18next";
import type { Entry } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { colorForBatch } from "./batch-color";
import { CellItemMenu } from "./cell-item-menu";
import { MealMenu } from "./meal-menu";
import { type CellSizePreset, minHeightClass } from "./mealplan-config";

export type EntryAction =
  | "replace"
  | "remove"
  | "view-detail"
  | "spread"
  | "split-batch"
  | "merge-batch";
export type MealAction = "clear" | "copy-days" | "reorder";

export type DragSource = { date: string; meal: string; position: number };
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

// D&D 転送用 MIME。同一ウィンドウ内でしか使わないが、application/json を明示すると
// 異エンジンでも parse エラーが起きにくい (Chromium の HTML5 D&D は text/plain を優先する)。
const DND_MIME = "application/json";

// 1 セル = 1 (date, meal)。0 品なら中央に `＋`、1..N 品ならレシピ名列 + 品ごと `⋯` +
// 右上 `＋` + 右下時間帯 `⋯`。品数超過は「他 N 件」で折り畳む。
// バッチ (batch_id) が複数行に跨る品には枠色 + 🔗 アイコンを描画する (spec の視覚識別要件)。
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
  const { t } = useTranslation();
  const limit = preset.visibleItems;
  const heightCls = minHeightClass(preset);
  const fontCls = preset.fontSizeClass;
  // 4 週表示のときは余白を切り詰める。それ以外は広めに取ってヒットボックスを確保する。
  const paddingCls = preset.periodWeeks === 4 ? "p-2" : "p-3";
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
      className={`flex ${heightCls} flex-col gap-1 rounded-md border border-neutral-200 bg-white ${paddingCls} ${dropOutlineCls}`}
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
        const color = colorForBatch(entry.batchId, entry.batchMemberCount);
        // batchMemberCount > 1 の作り置き行は個別移動を禁止する (design.md D1)。移動したい場合は
        // ⋯ → 「作り置きから外す」で単独行にしてから D&D する 2 段操作で扱う。
        const isBatchMember = entry.batchMemberCount > 1;
        const cursorCls = isBatchMember
          ? "cursor-not-allowed"
          : "cursor-grab active:cursor-grabbing";
        // ドラッグ中の source 行はフェードした placeholder として描画する (design.md D5)。
        // ghost 用の一時色は onDragStart で DOM に直接付けて browser にキャプチャさせ、rAF で剥がした
        // 後で setDragSource → 元の位置の source 行だけが薄い灰色に置き換わる (時間差の塗り分け):
        //   - ghost (掴んで動く方) = 薄い緑 (emerald-100 + emerald-400 ring)、視認しやすい
        //   - source (元の位置に残る方) = 灰色 + opacity-40 の placeholder、視点を奪わない
        const isDragSource =
          dragSource !== null &&
          dragSource.date === date &&
          dragSource.meal === meal &&
          dragSource.position === entry.position;
        const dragSourceCls = isDragSource ? "bg-neutral-100 opacity-40" : "";
        return (
          // biome-ignore lint/a11y/useSemanticElements: 品行の drag ハンドルは <li> にできない (レイアウト都合の div)
          <div
            key={entry.position}
            role="listitem"
            className={`flex items-center justify-between gap-1 rounded-sm border px-1 py-0.5 ${fontCls} text-neutral-800 ${cursorCls} ${dragSourceCls} ${
              color ? `${color}` : "border-transparent"
            }`}
            data-batch-id={entry.batchId}
            data-batch-member-count={entry.batchMemberCount}
            draggable={!isBatchMember}
            aria-label={isBatchMember ? undefined : t("pages.meal_plan.dnd.item_grabbable_label")}
            onDragStart={
              isBatchMember
                ? undefined
                : (e) => {
                    const source: DragSource = { date, meal, position: entry.position };
                    e.dataTransfer.effectAllowed = "move";
                    e.dataTransfer.setData(DND_MIME, JSON.stringify(source));
                    // browser の drag ghost キャプチャは onDragStart 完了直後に同期で走るため、
                    // ここで「ghost 用の一時色」を DOM に載せて capture させる。次 tick で剥がして
                    // source 側のフェード placeholder に置き換える。
                    // 結果: ghost = 薄い緑 (掴んで動く方) / source = 灰色フェード (元の位置に残る方)。
                    const target = e.currentTarget;
                    const ghostCls = ["bg-emerald-100", "ring-2", "ring-emerald-400"];
                    target.classList.add(...ghostCls);
                    requestAnimationFrame(() => {
                      target.classList.remove(...ghostCls);
                      onEntryDragStart(source);
                    });
                  }
            }
            onDragEnd={isBatchMember ? undefined : onEntryDragEnd}
            onDrop={
              isBatchMember
                ? undefined
                : (e) => {
                    e.stopPropagation();
                    handleCellDrop(e, entry.position);
                  }
            }
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
              onSpread={() => onEntryAction("spread", entry)}
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
