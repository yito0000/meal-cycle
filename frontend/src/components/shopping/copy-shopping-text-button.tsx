import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { writeClipboardText } from "@/lib/clipboard";
import type { Item } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";
import type { ShoppingPeriod } from "./shopping-period";
import { buildShoppingText } from "./shopping-text";

type Props = {
  items: Item[];
  period: ShoppingPeriod;
};

export function CopyShoppingTextButton({ items, period }: Props) {
  const { t } = useTranslation();
  const [writing, setWriting] = useState(false);
  const { text, count } = buildShoppingText(items, period, t);

  const onCopy = async () => {
    setWriting(true);
    try {
      await writeClipboardText(text);
      toast(t("pages.shopping.copy.copied", { count }));
    } catch {
      toast.error(t("pages.shopping.copy.failed"));
    } finally {
      setWriting(false);
    }
  };

  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      disabled={count === 0 || writing}
      onClick={onCopy}
      data-testid="shopping-copy-text"
    >
      {t("pages.shopping.copy.button")}
    </Button>
  );
}
