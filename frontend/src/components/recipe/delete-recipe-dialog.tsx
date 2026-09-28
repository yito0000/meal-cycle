import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

type Props = {
  open: boolean;
  recipeName: string;
  pending: boolean;
  error: boolean;
  onConfirm: () => void;
  onCancel: () => void;
};

export function DeleteRecipeDialog({
  open,
  recipeName,
  pending,
  error,
  onConfirm,
  onCancel,
}: Props) {
  const { t } = useTranslation();
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next && !pending) onCancel();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("pages.recipes.delete_confirm.title", { name: recipeName })}</DialogTitle>
          <DialogDescription>{t("pages.recipes.delete_confirm.body")}</DialogDescription>
        </DialogHeader>
        {error && (
          <p role="alert" className="text-sm text-red-700">
            {t("pages.recipes.delete_confirm.error")}
          </p>
        )}
        <DialogFooter className="gap-2">
          <Button type="button" variant="outline" disabled={pending} onClick={onCancel}>
            {t("pages.recipes.delete_confirm.cancel")}
          </Button>
          <Button type="button" variant="destructive" disabled={pending} onClick={onConfirm}>
            {t("pages.recipes.delete_confirm.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
