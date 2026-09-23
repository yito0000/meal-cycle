import { useTranslation } from "react-i18next";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";

interface Props {
  batchMemberCount: number;
  onReplace: () => void;
  onRemove: () => void;
  onViewDetail: () => void;
  onSpread: () => void;
  onSplitBatch: () => void;
  onMergeBatch: () => void;
}

// 1 品行の `⋯` メニュー: 入替 / 削除 / 詳細 / 作り置きを広げる / 作り置きから外す /
// 他の作り置きにまとめる。表示条件 (spec:「作り置きの外し / まとめ」Requirement 参照):
//   isInBatch = batchMemberCount > 1: 複数行の作り置きに属する
//   - 「作り置きから外す」は isInBatch のときだけ表示 (単独作り置きに外す対象はない)
//   - 「作り置きを広げる」はどの状態でも表示 (統合ダイアログで対象日 × 時間帯を選ぶ)
export function CellItemMenu({
  batchMemberCount,
  onReplace,
  onRemove,
  onViewDetail,
  onSpread,
  onSplitBatch,
  onMergeBatch,
}: Props) {
  const { t } = useTranslation();
  const isInBatch = batchMemberCount > 1;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={t("pages.meal_plan.cell_actions.item_menu")}
        className="px-1 text-neutral-400 hover:text-neutral-700"
      >
        ⋯
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={onReplace}>
          {t("pages.meal_plan.item_menu.replace")}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={onRemove}>
          {t("pages.meal_plan.item_menu.remove")}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={onViewDetail}>
          {t("pages.meal_plan.item_menu.view_detail")}
        </DropdownMenuItem>
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
