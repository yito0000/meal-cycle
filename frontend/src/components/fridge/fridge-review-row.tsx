import * as React from "react";
import { useTranslation } from "react-i18next";
import { useSettingsValues } from "@/components/settings/settings-gate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { TableCell, TableRow } from "@/components/ui/table";
import { formatQuantity } from "@/lib/format-quantity";
import { isImeComposing } from "@/lib/ime";
import { cn } from "@/lib/utils";
import {
  ExpiryStatus,
  type StockItem,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { formatDaysLeft } from "./days-left";
import { stockErrorOf } from "./error-messages";
import { ExpiryDot } from "./expiry-dot";
import { FieldError } from "./field-errors";
import { validateStockForm } from "./stock-form";
import { useUpdateStockAmount } from "./use-fridge";

type Props = {
  item: StockItem;
  reviewed: boolean;
  disabled: boolean;
  onOk: () => void;
  onReduced: () => void;
  onGone: () => void;
};

// 見直しの状態の 1 行。「減った」では量の欄だけを入力欄にし、打って Enter の 2 動作で終える (add-fridge-stock-review design.md D2)。
export function FridgeReviewRow({ item, reviewed, disabled, onOk, onReduced, onGone }: Props) {
  const { t } = useTranslation();
  const { measuringUnits } = useSettingsValues();
  const [amount, setAmount] = React.useState<string | null>(null);
  const [error, setError] = React.useState<string>();
  const update = useUpdateStockAmount();
  const none = t("pages.fridge.days_left.none");
  const expired = item.expiryStatus === ExpiryStatus.ExpiryExpired;

  const close = () => {
    setAmount(null);
    setError(undefined);
  };

  const save = () => {
    if (amount === null || update.isPending) return;
    const found = validateStockForm({
      name: item.name,
      amount,
      unit: item.unit,
      expiresAt: item.expiresAt,
    });
    if (found.amount) {
      // 0 は「ない」のつもりで打たれやすいので、検証のエラーではなく「ない」への案内を出す (design.md Risks)。
      setError(Number(amount) === 0 ? "pages.fridge.review.amount_zero" : found.amount);
      return;
    }
    update.mutate(
      { item, amount: Number(amount) },
      {
        onSuccess: () => {
          close();
          onReduced();
        },
        onError: (err) => setError(stockErrorOf(err).key),
      },
    );
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (isImeComposing(e)) return;
    if (e.key === "Enter") {
      e.preventDefault();
      save();
    } else if (e.key === "Escape") {
      e.preventDefault();
      close();
    }
  };

  return (
    <TableRow
      data-stock-id={item.id}
      data-reviewed={reviewed || undefined}
      className={cn(expired && "bg-red-50", reviewed && "opacity-50")}
    >
      <TableCell className="font-medium">
        <span className="flex items-center gap-2">
          <ExpiryDot status={item.expiryStatus} />
          <span>{item.name}</span>
        </span>
      </TableCell>
      <TableCell className="tabular-nums">
        {amount === null ? (
          formatQuantity(item.amount, item.unit, measuringUnits)
        ) : (
          <>
            <div className="flex items-center gap-1">
              <Input
                autoFocus
                type="number"
                inputMode="decimal"
                step="any"
                min="0"
                className="w-20"
                value={amount}
                onFocus={(e) => e.currentTarget.select()}
                onChange={(e) => {
                  setAmount(e.target.value);
                  setError(undefined);
                }}
                onKeyDown={onKeyDown}
                aria-label={t("pages.fridge.field.amount")}
                aria-invalid={!!error}
              />
              <span className="text-sm text-neutral-700">{item.unit}</span>
            </div>
            <FieldError messageKey={error} />
          </>
        )}
      </TableCell>
      <TableCell className="tabular-nums text-neutral-700">{item.expiresAt || none}</TableCell>
      <TableCell className={cn("tabular-nums", expired ? "text-red-700" : "text-neutral-700")}>
        {formatDaysLeft(t, item.daysLeft)}
      </TableCell>
      <TableCell>
        <div className="flex gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={disabled}
            onClick={onOk}
            data-review-ok={item.id}
          >
            {t("pages.fridge.review.ok")}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={disabled}
            onClick={() => {
              setAmount(String(item.amount));
              setError(undefined);
            }}
          >
            {t("pages.fridge.review.reduced")}
          </Button>
          <Button type="button" variant="ghost" size="sm" disabled={disabled} onClick={onGone}>
            {t("pages.fridge.review.gone")}
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}
