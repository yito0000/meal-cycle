import type * as React from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";

type Props = {
  done: number;
  total: number;
  onFinish: () => void;
  finishRef?: React.Ref<HTMLButtonElement>;
};

export function FridgeReviewBar({ done, total, onFinish, finishRef }: Props) {
  const { t } = useTranslation();
  return (
    <div className="flex items-center justify-between gap-4 rounded-md border border-blue-200 bg-blue-50 px-4 py-2">
      <p className="text-sm text-neutral-800" aria-live="polite">
        {t("pages.fridge.review.progress", { done, total })}
      </p>
      <Button ref={finishRef} type="button" variant="outline" size="sm" onClick={onFinish}>
        {t("pages.fridge.review.finish")}
      </Button>
    </div>
  );
}
