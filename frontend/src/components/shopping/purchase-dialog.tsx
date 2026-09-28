import { useState } from "react";
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
import { Input } from "@/components/ui/input";
import type { Item } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";
import { stockErrorOf } from "../fridge/error-messages";
import { FieldError } from "../fridge/field-errors";
import type { StockField, StockFormErrors, StockFormValues } from "../fridge/stock-form";
import { toPurchaseRows, validatePurchaseRows } from "./purchase-form";
import { useRegisterPurchase } from "./use-purchase";

type Props = {
  open: boolean;
  items: Item[];
  onOpenChange: (open: boolean) => void;
  onRegistered: (count: number) => void;
};

export function PurchaseDialog({ open, items, onOpenChange, onRegistered }: Props) {
  const { t } = useTranslation();
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] max-w-3xl flex-col">
        <DialogHeader>
          <DialogTitle>{t("pages.shopping.purchase.dialog_title")}</DialogTitle>
          <DialogDescription>{t("pages.shopping.purchase.dialog_description")}</DialogDescription>
        </DialogHeader>
        {/* 閉じると DialogContent ごとアンマウントされるので、開くたびに初期値から作り直される (design.md D7)。 */}
        <PurchaseForm
          items={items}
          onCancel={() => onOpenChange(false)}
          onRegistered={onRegistered}
        />
      </DialogContent>
    </Dialog>
  );
}

function PurchaseForm({
  items,
  onCancel,
  onRegistered,
}: {
  items: Item[];
  onCancel: () => void;
  onRegistered: (count: number) => void;
}) {
  const { t } = useTranslation();
  const [rows, setRows] = useState<StockFormValues[]>(() => toPurchaseRows(items));
  const [errors, setErrors] = useState<StockFormErrors[]>([]);
  const [formError, setFormError] = useState<string>();
  const register = useRegisterPurchase();

  const set = (index: number, field: StockField) => (value: string) => {
    setRows((rs) => rs.map((r, i) => (i === index ? { ...r, [field]: value } : r)));
    setErrors((es) => es.map((e, i) => (i === index ? { ...e, [field]: undefined } : e)));
  };

  const submit = () => {
    if (register.isPending) return;
    const found = validatePurchaseRows(rows);
    setErrors(found.errors);
    setFormError(undefined);
    if (found.hasError) return;
    register.mutate(rows, {
      onSuccess: () => onRegistered(rows.length),
      onError: (err) => setFormError(stockErrorOf(err).key),
    });
  };

  return (
    <>
      {formError && (
        <p role="alert" className="text-sm text-red-700">
          {t(formError)}
        </p>
      )}
      <div className="-mx-1 min-h-0 flex-1 overflow-y-auto px-1">
        <div className="grid grid-cols-[1fr_6rem_6rem_11rem] gap-x-2 gap-y-1 text-xs text-neutral-500">
          <span>{t("pages.fridge.field.name")}</span>
          <span>{t("pages.fridge.field.amount")}</span>
          <span>{t("pages.fridge.field.unit")}</span>
          <span>{t("pages.fridge.field.expires_at")}</span>
        </div>
        <ul className="mt-1 flex flex-col gap-2">
          {rows.map((row, i) => {
            const e = errors[i] ?? {};
            return (
              <li
                // 行の追加と削除が無く、並びも開いた時点で固定なので index で足りる。
                // biome-ignore lint/suspicious/noArrayIndexKey: 行の順序と数は開いている間変わらない。
                key={i}
                data-testid="purchase-row"
                className="grid grid-cols-[1fr_6rem_6rem_11rem] items-start gap-x-2"
              >
                <div>
                  <Input
                    value={row.name}
                    onChange={(ev) => set(i, "name")(ev.target.value)}
                    aria-label={t("pages.fridge.field.name")}
                    aria-invalid={!!e.name}
                  />
                  <FieldError messageKey={e.name} />
                </div>
                <div>
                  <Input
                    inputMode="decimal"
                    value={row.amount}
                    onChange={(ev) => set(i, "amount")(ev.target.value)}
                    aria-label={t("pages.fridge.field.amount")}
                    aria-invalid={!!e.amount}
                  />
                  <FieldError messageKey={e.amount} />
                </div>
                <div>
                  <Input
                    value={row.unit}
                    onChange={(ev) => set(i, "unit")(ev.target.value)}
                    aria-label={t("pages.fridge.field.unit")}
                    aria-invalid={!!e.unit}
                  />
                  <FieldError messageKey={e.unit} />
                </div>
                <div>
                  {/* DatePicker は空の値を持てないため、S06 と同じネイティブの日付入力にする (design.md D5)。 */}
                  <Input
                    type="date"
                    value={row.expiresAt}
                    onChange={(ev) => set(i, "expiresAt")(ev.target.value)}
                    aria-label={t("pages.fridge.field.expires_at")}
                    aria-invalid={!!e.expiresAt}
                  />
                  <FieldError messageKey={e.expiresAt} />
                </div>
              </li>
            );
          })}
        </ul>
      </div>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onCancel}>
          {t("pages.shopping.purchase.cancel")}
        </Button>
        <Button type="button" onClick={submit} disabled={register.isPending}>
          {t("pages.shopping.purchase.submit")}
        </Button>
      </DialogFooter>
    </>
  );
}
