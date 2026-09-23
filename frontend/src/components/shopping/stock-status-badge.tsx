import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { StockStatus } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";

const STYLE: Partial<Record<StockStatus, string>> = {
  [StockStatus.StockNone]: "border-neutral-300 bg-white text-neutral-800",
  [StockStatus.StockPartial]: "border-amber-300 bg-amber-50 text-amber-900",
  [StockStatus.StockCovered]: "border-emerald-200 bg-emerald-50 text-emerald-800",
};

const LABEL_KEY: Partial<Record<StockStatus, string>> = {
  [StockStatus.StockNone]: "pages.shopping.stock_status.none",
  [StockStatus.StockPartial]: "pages.shopping.stock_status.partial",
  [StockStatus.StockCovered]: "pages.shopping.stock_status.covered",
};

// 状態は色だけに頼らず文字でも示す (spec「在庫の状態表示」)。
export function StockStatusBadge({ status }: { status: StockStatus }) {
  const { t } = useTranslation();
  const label = LABEL_KEY[status];
  if (!label) return null;
  return (
    <span
      data-stock-status={status}
      className={cn(
        "inline-flex shrink-0 items-center rounded border px-1.5 py-0.5 text-xs leading-none",
        STYLE[status],
      )}
    >
      {t(label)}
    </span>
  );
}
