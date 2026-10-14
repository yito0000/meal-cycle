import { useTranslation } from "react-i18next";
import { Button } from "../ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";

interface Props {
  open: boolean;
  removedCount: number;
  onConfirm: () => void;
  onCancel: () => void;
}

// 「作り置きをやめる」は表示中の期間外の行も消すので、事後の「元に戻す」ではなく事前に確かめる
// (revise-mealplan-destructive-actions design.md D3)。
export function DissolveBatchDialog({ open, removedCount, onConfirm, onCancel }: Props) {
  const { t } = useTranslation();
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onCancel();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("pages.meal_plan.dissolve.title")}</DialogTitle>
          <DialogDescription>
            {t("pages.meal_plan.dissolve.description", { count: removedCount })}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={onCancel}>
            {t("pages.meal_plan.dissolve.cancel")}
          </Button>
          <Button variant="destructive" onClick={onConfirm}>
            {t("pages.meal_plan.dissolve.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
