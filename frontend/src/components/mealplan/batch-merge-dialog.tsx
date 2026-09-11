import { useTranslation } from "react-i18next";
import type { BatchSummary } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { Button } from "../ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "../ui/dialog";

interface Props {
  open: boolean;
  candidates: BatchSummary[];
  onOpenChange: (open: boolean) => void;
  onSelect: (batchID: string) => void;
}

// 「バッチに合流する」を押したときの候補一覧。同じ slot の他バッチだけを表示する
// (spec の合流候補要件、design.md D7)。候補ゼロなら空メッセージを表示。
export function BatchMergeDialog({ open, candidates, onOpenChange, onSelect }: Props) {
  const { t } = useTranslation();
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("pages.meal_plan.batch.merge_dialog.title")}</DialogTitle>
        </DialogHeader>
        {candidates.length === 0 ? (
          <p className="text-sm text-neutral-600">
            {t("pages.meal_plan.batch.merge_dialog.empty")}
          </p>
        ) : (
          <ul className="flex max-h-64 flex-col gap-1 overflow-y-auto">
            {candidates.map((c) => (
              <li key={c.batchId}>
                <button
                  type="button"
                  className="block w-full cursor-pointer rounded-md border border-neutral-200 px-3 py-2 text-left text-sm hover:bg-neutral-50"
                  onClick={() => onSelect(c.batchId)}
                >
                  {t("pages.meal_plan.batch.merge_dialog.item_label", {
                    date: c.date,
                    recipeName: c.recipeName,
                    count: c.memberCount,
                  })}
                </button>
              </li>
            ))}
          </ul>
        )}
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("pages.meal_plan.copy.conflict.cancel")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
