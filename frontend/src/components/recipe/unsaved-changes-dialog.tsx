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
  onDiscard: () => void;
  onKeepEditing: () => void;
};

export function UnsavedChangesDialog({ open, onDiscard, onKeepEditing }: Props) {
  const { t } = useTranslation();
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onKeepEditing();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("pages.recipes.edit.unsaved_confirm.title")}</DialogTitle>
          <DialogDescription>{t("pages.recipes.edit.unsaved_confirm.body")}</DialogDescription>
        </DialogHeader>
        <DialogFooter className="gap-2">
          <Button type="button" variant="outline" onClick={onKeepEditing}>
            {t("pages.recipes.edit.unsaved_confirm.keep_editing")}
          </Button>
          <Button type="button" variant="destructive" onClick={onDiscard}>
            {t("pages.recipes.edit.unsaved_confirm.discard")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
