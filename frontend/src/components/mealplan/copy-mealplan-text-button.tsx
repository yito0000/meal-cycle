import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { writeClipboardText } from "@/lib/clipboard";
import type { Entry } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { buildMealPlanText } from "./mealplan-text";

type Props = {
  entries: readonly Entry[];
  period: { start: string; end: string };
};

export function CopyMealPlanTextButton({ entries, period }: Props) {
  const { t, i18n } = useTranslation();
  const [writing, setWriting] = useState(false);

  const onCopy = async () => {
    const { text, dayCount } = buildMealPlanText(entries, period, t, i18n.language);
    setWriting(true);
    try {
      await writeClipboardText(text);
      toast(t("pages.meal_plan.text_copy.copied", { count: dayCount }));
    } catch {
      toast.error(t("pages.meal_plan.text_copy.failed"));
    } finally {
      setWriting(false);
    }
  };

  return (
    <Button
      type="button"
      variant="outline"
      disabled={entries.length === 0 || writing}
      onClick={onCopy}
      data-testid="mealplan-copy-text"
    >
      {t("pages.meal_plan.text_copy.button")}
    </Button>
  );
}
