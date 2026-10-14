import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { Entry } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { colorForBatch } from "./batch-color";
import { CellItemMenu } from "./cell-item-menu";
import type { DragSource, EntryAction } from "./meal-plan-cell";

// D&D 転送用 MIME。同一ウィンドウ内でしか使わないが、application/json を明示すると
// 異エンジンでも parse エラーが起きにくい (Chromium の HTML5 D&D は text/plain を優先する)。
export const DND_MIME = "application/json";

interface Props {
  entry: Entry;
  cellItemCount: number;
  fontClass: string;
  draggable: boolean;
  isDragSource: boolean;
  onAction: (action: EntryAction, entry: Entry) => void;
  onDragStart?: (source: DragSource) => void;
  onDragEnd?: () => void;
  onDropOnRow?: (e: React.DragEvent<HTMLDivElement>, position: number) => void;
}

// セルの中と「他 N 件」の一覧の両方で使う品行 (add-mealplan-cell-overflow-list design.md D3)。
export function MealItemRow({
  entry,
  cellItemCount,
  fontClass,
  draggable,
  isDragSource,
  onAction,
  onDragStart,
  onDragEnd,
  onDropOnRow,
}: Props) {
  const { t } = useTranslation();
  const [menuOpen, setMenuOpen] = useState(false);
  const nameButtonRef = useRef<HTMLButtonElement | null>(null);
  // チェックボックスに draggable={false} を付けても、押したまま動かすと親の品行のドラッグが始まる。
  // 押し下げた場所を覚えておき、dragstart 側で取り消す (revise-mealplan-item-row design.md D2)。
  const pressedCheckboxRef = useRef(false);
  const color = colorForBatch(entry.batchId, entry.batchMemberCount);
  // ドラッグ中の source 行はフェードした placeholder として描画する (design.md D5)。
  // ghost 用の一時色は onDragStart で DOM に直接付けて browser にキャプチャさせ、rAF で剥がした
  // 後で setDragSource → 元の位置の source 行だけが薄い灰色に置き換わる (時間差の塗り分け):
  //   - ghost (掴んで動く方) = 薄い緑 (emerald-100 + emerald-400 ring)、視認しやすい
  //   - source (元の位置に残る方) = 灰色 + opacity-40 の placeholder、視点を奪わない
  const dragSourceCls = isDragSource ? "bg-neutral-100 opacity-40" : "";

  return (
    // biome-ignore lint/a11y/useSemanticElements: 品行の drag ハンドルは <li> にできない (レイアウト都合の div)
    <div
      role="listitem"
      className={`relative flex cursor-pointer items-start gap-1 rounded-sm border px-0.5 py-0.5 hover:bg-blue-100 ${fontClass} text-neutral-800 ${dragSourceCls} ${
        color ? `${color}` : "border-transparent"
      }`}
      data-batch-id={entry.batchId}
      data-batch-member-count={entry.batchMemberCount}
      data-cooked={entry.cooked}
      draggable={draggable}
      onMouseDown={(e) => {
        pressedCheckboxRef.current = e.target instanceof HTMLInputElement;
      }}
      aria-label={draggable ? t("pages.meal_plan.dnd.item_grabbable_label") : undefined}
      onDragStart={
        draggable
          ? (e) => {
              if (pressedCheckboxRef.current) {
                e.preventDefault();
                return;
              }
              const source: DragSource = {
                id: entry.id,
                date: entry.date,
                meal: entry.meal,
                position: entry.position,
              };
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
                onDragStart?.(source);
              });
            }
          : undefined
      }
      onDragEnd={draggable ? onDragEnd : undefined}
      onDrop={
        draggable && onDropOnRow
          ? (e) => {
              e.stopPropagation();
              onDropOnRow(e, entry.position);
            }
          : undefined
      }
    >
      <input
        type="checkbox"
        className="mt-[0.2em] size-[0.9em] shrink-0 cursor-pointer accent-emerald-600"
        aria-label={t("pages.meal_plan.cell.cooked")}
        checked={entry.cooked}
        onChange={() => onAction("toggle-cooked", entry)}
      />
      {/* click はドラッグが始まると発火しないので、クリックとドラッグをここで区別できる (design.md D1)。 */}
      <button
        ref={nameButtonRef}
        type="button"
        className="min-w-0 flex-1 cursor-pointer text-left"
        aria-haspopup="menu"
        aria-expanded={menuOpen}
        onClick={() => setMenuOpen(true)}
      >
        <span className="line-clamp-2 [overflow-wrap:anywhere]">{entry.recipeName}</span>
      </button>
      <CellItemMenu
        open={menuOpen}
        onOpenChange={setMenuOpen}
        onCloseAutoFocus={(e) => {
          e.preventDefault();
          nameButtonRef.current?.focus();
        }}
        batchMemberCount={entry.batchMemberCount}
        position={entry.position}
        cellItemCount={cellItemCount}
        onReplace={() => onAction("replace", entry)}
        onRemove={() => onAction("remove", entry)}
        onViewDetail={() => onAction("view-detail", entry)}
        onMoveUp={() => onAction("move-up", entry)}
        onMoveDown={() => onAction("move-down", entry)}
        onSpread={() => onAction("spread", entry)}
        onSplitBatch={() => onAction("split-batch", entry)}
        onMergeBatch={() => onAction("merge-batch", entry)}
      />
    </div>
  );
}
