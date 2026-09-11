import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "../ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "../ui/dialog";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelect: (shareBatch: boolean) => void;
}

// 連続 N 日コピー実行前に「作り置き (1 バッチ共有) / 別調理 (N バッチ)」を選ばせる。
// 既定は「作り置き」(ペルソナと一致)。onSelect(shareBatch) で呼び出し元に返す。
export function CopyModeDialog({ open, onOpenChange, onSelect }: Props) {
  const { t } = useTranslation();
  const [shareBatch, setShareBatch] = useState(true);

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        onOpenChange(next);
        if (!next) setShareBatch(true);
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("pages.meal_plan.copy.mode.title")}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-3 text-sm">
          <label className="flex items-start gap-3 rounded-md border border-neutral-200 p-3 hover:bg-neutral-50">
            <input
              type="radio"
              name="copy-mode"
              checked={shareBatch}
              onChange={() => setShareBatch(true)}
              className="mt-1"
            />
            <span className="flex flex-col">
              <span className="font-medium text-neutral-900">
                {t("pages.meal_plan.copy.mode.leftover")}
              </span>
              <span className="text-xs text-neutral-500">
                {t("pages.meal_plan.copy.mode.leftover_hint")}
              </span>
            </span>
          </label>
          <label className="flex items-start gap-3 rounded-md border border-neutral-200 p-3 hover:bg-neutral-50">
            <input
              type="radio"
              name="copy-mode"
              checked={!shareBatch}
              onChange={() => setShareBatch(false)}
              className="mt-1"
            />
            <span className="flex flex-col">
              <span className="font-medium text-neutral-900">
                {t("pages.meal_plan.copy.mode.distinct")}
              </span>
              <span className="text-xs text-neutral-500">
                {t("pages.meal_plan.copy.mode.distinct_hint")}
              </span>
            </span>
          </label>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("pages.meal_plan.copy.conflict.cancel")}
          </Button>
          <Button
            onClick={() => {
              onSelect(shareBatch);
              setShareBatch(true);
            }}
          >
            {t("pages.meal_plan.copy.mode.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
