import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { ExpiryStatus } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";

const DOT_CLASS: Partial<Record<ExpiryStatus, string>> = {
  [ExpiryStatus.ExpiryUrgent]: "bg-red-500",
  [ExpiryStatus.ExpirySoon]: "bg-amber-400",
  [ExpiryStatus.ExpiryFresh]: "bg-emerald-500",
};

const LABEL_KEY: Partial<Record<ExpiryStatus, string>> = {
  [ExpiryStatus.ExpiryUrgent]: "pages.fridge.expiry_status.urgent",
  [ExpiryStatus.ExpirySoon]: "pages.fridge.expiry_status.soon",
  [ExpiryStatus.ExpiryFresh]: "pages.fridge.expiry_status.fresh",
};

export function ExpiryDot({ status }: { status: ExpiryStatus }) {
  const { t } = useTranslation();
  // 期限切れは 🔴 と色だけで区別させず、ドットと形の違う文字バッジで示す (design.md D14)。
  if (status === ExpiryStatus.ExpiryExpired) {
    return (
      <span
        data-expiry-status={status}
        className="inline-flex shrink-0 items-center rounded border border-red-700 bg-red-700 px-1.5 py-0.5 text-xs font-semibold leading-none text-white"
      >
        {t("pages.fridge.expiry_status.expired")}
      </span>
    );
  }
  const dot = DOT_CLASS[status];
  const label = LABEL_KEY[status];
  if (!dot || !label) return null;
  return (
    <span
      role="img"
      aria-label={t(label)}
      data-expiry-status={status}
      className={cn("inline-block h-2.5 w-2.5 shrink-0 rounded-full", dot)}
    />
  );
}
