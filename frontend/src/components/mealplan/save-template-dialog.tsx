import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { FieldError } from "@/components/fridge/field-errors";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { isImeComposing } from "@/lib/ime";
import { errorI18nKey } from "./error-messages";
import { useSaveTemplate } from "./use-templates";

// Go の検証 (mealplan.TemplateNameMaxRunes) と揃える。最終防衛線は Go 側。
const NAME_MAX = 50;

type Props = {
  open: boolean;
  weekStart: string;
  onOpenChange: (open: boolean) => void;
};

export function SaveTemplateDialog({ open, weekStart, onOpenChange }: Props) {
  const { t } = useTranslation();
  const [name, setName] = useState("");
  const [error, setError] = useState<string>();
  const save = useSaveTemplate();

  const close = (next: boolean) => {
    onOpenChange(next);
    if (!next) {
      setName("");
      setError(undefined);
    }
  };

  const submit = () => {
    if (save.isPending) return;
    const trimmed = name.trim();
    if (trimmed === "") {
      setError("pages.meal_plan.errors.template_name_empty");
      return;
    }
    if ([...trimmed].length > NAME_MAX) {
      setError("pages.meal_plan.errors.template_name_too_long");
      return;
    }
    save.mutate(
      { weekStart, name: trimmed },
      {
        onSuccess: () => {
          toast(t("pages.meal_plan.template.saved", { name: trimmed }));
          close(false);
        },
        onError: (err) => setError(errorI18nKey(err)),
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t("pages.meal_plan.template.save_title")}</DialogTitle>
          <DialogDescription>{t("pages.meal_plan.template.save_description")}</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="template-name">{t("pages.meal_plan.template.name_label")}</Label>
          <Input
            id="template-name"
            autoFocus
            value={name}
            maxLength={NAME_MAX * 2}
            onChange={(e) => {
              setName(e.target.value);
              setError(undefined);
            }}
            onKeyDown={(e) => {
              if (e.key !== "Enter" || isImeComposing(e)) return;
              e.preventDefault();
              submit();
            }}
            aria-invalid={!!error}
          />
          <FieldError messageKey={error} />
        </div>
        <DialogFooter className="gap-2">
          <Button type="button" variant="outline" onClick={() => close(false)}>
            {t("pages.meal_plan.template.cancel")}
          </Button>
          <Button type="button" disabled={save.isPending} onClick={submit}>
            {t("pages.meal_plan.template.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
