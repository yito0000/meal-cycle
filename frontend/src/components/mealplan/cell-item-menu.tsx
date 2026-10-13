import { useTranslation } from "react-i18next";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCloseAutoFocus: (e: Event) => void;
  batchMemberCount: number;
  position: number;
  cellItemCount: number;
  onReplace: () => void;
  onRemove: () => void;
  onViewDetail: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onSpread: () => void;
  onSplitBatch: () => void;
  onMergeBatch: () => void;
}

// 1 品の品のメニュー: 入替 / 削除 / 詳細 / 上へ移動 / 下へ移動 / 作り置きを広げる / 作り置きから外す / 他の作り置きにまとめる。
// 開閉は品行のレシピ名のボタンが持ち、ここでは `relative` な品行に重なる透明な基準要素と中身だけを描く。
// 表示条件 (spec:「作り置きの外し / まとめ」Requirement 参照):
//   isInBatch = batchMemberCount > 1: 複数行の作り置きに属する
//   - 「作り置きから外す」は isInBatch のときだけ表示 (単独作り置きに外す対象はない)
//   - 「作り置きを広げる」はどの状態でも表示 (統合ダイアログで対象日 × 時間帯を選ぶ)
//   - 「上へ移動」「下へ移動」はセルに 2 品以上あるときだけ表示。端の品では消さずに disabled にして
//     項目の位置を揃える (revise-mealplan-reorder-menu design.md D2)
export function CellItemMenu({
  open,
  onOpenChange,
  onCloseAutoFocus,
  batchMemberCount,
  position,
  cellItemCount,
  onReplace,
  onRemove,
  onViewDetail,
  onMoveUp,
  onMoveDown,
  onSpread,
  onSplitBatch,
  onMergeBatch,
}: Props) {
  const { t } = useTranslation();
  const isInBatch = batchMemberCount > 1;
  const showMove = cellItemCount >= 2;
  return (
    <DropdownMenu open={open} onOpenChange={onOpenChange}>
      {/* Radix の Trigger は pointerdown で開くので、ドラッグと区別できない。pointer-events を切って
          位置の基準にだけ使う (revise-mealplan-item-row design.md D1)。 */}
      <DropdownMenuTrigger asChild>
        <span aria-hidden="true" tabIndex={-1} className="pointer-events-none absolute inset-0" />
      </DropdownMenuTrigger>
      <DropdownMenuContent side="right" align="start" onCloseAutoFocus={onCloseAutoFocus}>
        <DropdownMenuItem onClick={onReplace}>
          {t("pages.meal_plan.item_menu.replace")}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={onRemove}>
          {t("pages.meal_plan.item_menu.remove")}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={onViewDetail}>
          {t("pages.meal_plan.item_menu.view_detail")}
        </DropdownMenuItem>
        {showMove && (
          <>
            <DropdownMenuItem disabled={position === 0} onClick={onMoveUp}>
              {t("pages.meal_plan.item_menu.move_up")}
            </DropdownMenuItem>
            <DropdownMenuItem disabled={position === cellItemCount - 1} onClick={onMoveDown}>
              {t("pages.meal_plan.item_menu.move_down")}
            </DropdownMenuItem>
          </>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={onSpread}>
          {t("pages.meal_plan.item_menu.spread")}
        </DropdownMenuItem>
        {isInBatch && (
          <DropdownMenuItem onClick={onSplitBatch}>
            {t("pages.meal_plan.item_menu.split_batch")}
          </DropdownMenuItem>
        )}
        <DropdownMenuItem onClick={onMergeBatch}>
          {t("pages.meal_plan.item_menu.merge_batch")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
