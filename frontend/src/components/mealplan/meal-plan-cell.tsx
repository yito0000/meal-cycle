import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { Entry } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { colorForBatch } from "./batch-color";
import { CellItemMenu } from "./cell-item-menu";
import { MealMenu } from "./meal-menu";
import { type CellSizePreset, minHeightClass } from "./mealplan-config";

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

// 1 セル = 1 (date, meal)。0 品なら中央に `＋`、1..N 品なら品行 (「作った」チェックボックス + レシピ名) +
// 右上 `＋` + 右下時間帯 `⋯`。品数超過は「他 N 件」で折り畳む。
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
  const { t } = useTranslation();
  const limit = preset.visibleItems;
  const heightCls = minHeightClass(preset);
  const fontCls = preset.fontSizeClass;
  const [menuPosition, setMenuPosition] = useState<number | null>(null);
  const menuButtonRef = useRef<HTMLButtonElement | null>(null);
  // チェックボックスに draggable={false} を付けても、押したまま動かすと親の品行のドラッグが始まる。
  // 押し下げた場所を覚えておき、dragstart 側で取り消す (revise-mealplan-item-row design.md D2)。
  const pressedCheckboxRef = useRef(false);
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
        const color = colorForBatch(entry.batchId, entry.batchMemberCount);
        // batchMemberCount > 1 の作り置き行は個別移動を禁止する (design.md D1)。移動したい場合は
        // 品のメニューの「作り置きから外す」で単独行にしてから D&D する 2 段操作で扱う。
        const isBatchMember = entry.batchMemberCount > 1;
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
            className={`relative flex cursor-pointer items-start gap-1 rounded-sm border px-0.5 py-0.5 hover:bg-blue-100 ${fontCls} text-neutral-800 ${dragSourceCls} ${
              color ? `${color}` : "border-transparent"
            }`}
            data-batch-id={entry.batchId}
            data-batch-member-count={entry.batchMemberCount}
            data-cooked={entry.cooked}
            draggable={!isBatchMember}
            onMouseDown={(e) => {
              pressedCheckboxRef.current = e.target instanceof HTMLInputElement;
            }}
            aria-label={isBatchMember ? undefined : t("pages.meal_plan.dnd.item_grabbable_label")}
            onDragStart={
              isBatchMember
                ? undefined
                : (e) => {
                    if (pressedCheckboxRef.current) {
                      e.preventDefault();
                      return;
                    }
                    const source: DragSource = { date, meal, position: entry.position };
                    e.dataTransfer.effectAllowed = "move";
                    e.dataTransfer.setData(DND_MIME, JSON.stringify(source));
                    // browser の drag ghost キャプチャは onDragStart 完了直後に同期で走るため、
                    // ここで「ghost 用の一時色」を DOM に載せて capture させる。次 tick で剥がして
                    // source 側のフェード placeholder に置き換える。
                    // 結果: ghost = 薄い緑 (掴んで動く方) / source = 灰色フェード (元の位置に残る方)。
                    const target = e.currentTarget;
                    // 掴んだ時点の品行は hover 中なので、hover の背景色に負けないよう important にする。
                    const ghostCls = ["bg-emerald-100!", "ring-2", "ring-emerald-400"];
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
            <input
              type="checkbox"
              className="mt-[0.2em] size-[0.9em] shrink-0 cursor-pointer accent-emerald-600"
              aria-label={t("pages.meal_plan.cell.cooked")}
              checked={entry.cooked}
              onChange={() => onEntryAction("toggle-cooked", entry)}
            />
            {/* click はドラッグが始まると発火しないので、クリックとドラッグをここで区別できる (design.md D1)。 */}
            <button
              type="button"
              className="min-w-0 flex-1 cursor-pointer text-left"
              aria-haspopup="menu"
              aria-expanded={menuPosition === entry.position}
              onClick={(e) => {
                menuButtonRef.current = e.currentTarget;
                setMenuPosition(entry.position);
              }}
            >
              <span className="line-clamp-2 [overflow-wrap:anywhere]">{entry.recipeName}</span>
            </button>
            <CellItemMenu
              open={menuPosition === entry.position}
              onOpenChange={(next) => setMenuPosition(next ? entry.position : null)}
              onCloseAutoFocus={(e) => {
                e.preventDefault();
                menuButtonRef.current?.focus();
              }}
              batchMemberCount={entry.batchMemberCount}
              position={entry.position}
              cellItemCount={entries.length}
              onReplace={() => onEntryAction("replace", entry)}
              onRemove={() => onEntryAction("remove", entry)}
              onViewDetail={() => onEntryAction("view-detail", entry)}
              onMoveUp={() => onEntryAction("move-up", entry)}
              onMoveDown={() => onEntryAction("move-down", entry)}
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
        />
      </div>
    </div>
  );
}
