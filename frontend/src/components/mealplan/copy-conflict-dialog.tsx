import { useTranslation } from "react-i18next";
import { Button } from "../ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "../ui/dialog";

export type CopyConflictMode = "append" | "overwrite" | "cancel";

interface Props {
  open: boolean;
  conflictTargets: string[];
  onSelect: (mode: CopyConflictMode) => void;
}

// 作り置きコピー先に既存割当がある場合の 3 択ダイアログ。
export function CopyConflictDialog({ open, conflictTargets, onSelect }: Props) {
  const { t } = useTranslation();
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onSelect("cancel");
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("pages.meal_plan.copy.conflict.title")}</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-neutral-700">
          {t("pages.meal_plan.copy.conflict.body", { count: conflictTargets.length })}
        </p>
        <ul className="max-h-32 overflow-y-auto rounded-md border border-neutral-200 p-2 text-xs text-neutral-600">
          {conflictTargets.map((d) => (
            <li key={d}>{d}</li>
          ))}
        </ul>
        <DialogFooter>
          <Button variant="outline" onClick={() => onSelect("cancel")}>
            {t("pages.meal_plan.copy.conflict.cancel")}
          </Button>
          <Button variant="outline" onClick={() => onSelect("append")}>
            {t("pages.meal_plan.copy.conflict.append")}
          </Button>
          <Button onClick={() => onSelect("overwrite")}>
            {t("pages.meal_plan.copy.conflict.overwrite")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
