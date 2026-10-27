import { useQueryClient } from "@tanstack/react-query";
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
import { formatDisplayDate } from "@/lib/format-date";
import { formatQuantity } from "@/lib/format-quantity";
import type { ConsumptionPlan } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { FridgeService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { stockErrorOf } from "../fridge/error-messages";
import { FieldError } from "../fridge/field-errors";
import { useSettingsValues } from "../settings/settings-gate";
import {
  type ConsumeRow,
  toConsumeRows,
  toConsumptions,
  validateConsumeRows,
} from "./consume-form";

type Props = {
  plan: ConsumptionPlan | null;
  onClose: () => void;
  onConsumed: (count: number) => void;
};

// 印は開く前に付いているので、閉じる操作 (Esc、×、「減らさない」) はすべて在庫を変えないことと同じ意味になる
// (add-mealplan-cook-consume design.md D5)。
export function ConsumeStockDialog({ plan, onClose, onConsumed }: Props) {
  const { t } = useTranslation();
  return (
    <Dialog
      open={plan !== null}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent className="flex max-h-[85vh] max-w-2xl flex-col">
        <DialogHeader>
          <DialogTitle>{t("pages.meal_plan.consume.title")}</DialogTitle>
          <DialogDescription>{t("pages.meal_plan.consume.description")}</DialogDescription>
        </DialogHeader>
        {plan && <ConsumeForm plan={plan} onClose={onClose} onConsumed={onConsumed} />}
      </DialogContent>
    </Dialog>
  );
}

function ConsumeForm({
  plan,
  onClose,
  onConsumed,
}: {
  plan: ConsumptionPlan;
  onClose: () => void;
  onConsumed: (count: number) => void;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { measuringUnits } = useSettingsValues();
  const [rows, setRows] = useState<ConsumeRow[]>(() => toConsumeRows(plan));
  const [errors, setErrors] = useState<(string | undefined)[]>([]);
  const [formError, setFormError] = useState<string>();
  const [pending, setPending] = useState(false);

  const setAmount = (index: number, value: string) => {
    setRows((rs) => rs.map((r, i) => (i === index ? { ...r, amount: value } : r)));
    setErrors((es) => es.map((e, i) => (i === index ? undefined : e)));
  };

  const submit = async () => {
    if (pending) return;
    const found = validateConsumeRows(rows);
    setErrors(found.errors);
    setFormError(undefined);
    if (found.hasError) return;
    const items = toConsumptions(rows);
    setPending(true);
    try {
      await FridgeService.Consume(items);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["fridge"] }),
        queryClient.invalidateQueries({ queryKey: ["shopping"] }),
      ]);
      onConsumed(items.length);
    } catch (err) {
      setFormError(stockErrorOf(err).key);
    } finally {
      setPending(false);
    }
  };

  let offset = 0;
  return (
    <>
      {formError && (
        <p role="alert" className="text-sm text-red-700">
          {t(formError)}
        </p>
      )}
      <div className="-mx-1 min-h-0 flex-1 overflow-y-auto px-1">
        <div className="flex flex-col gap-4">
          {(plan.lines ?? []).map((line) => {
            const start = offset;
            const candidates = line.candidates ?? [];
            offset += candidates.length;
            return (
              <section
                key={`${line.name}\u0000${line.unit}`}
                aria-label={line.name}
                data-testid="consume-line"
              >
                <h3 className="text-sm font-semibold">
                  {line.name}
                  <span className="ml-2 text-xs font-normal text-neutral-500">
                    {formatQuantity(line.requiredAmount, line.unit, measuringUnits)}
                  </span>
                </h3>
                <div className="mt-1 grid grid-cols-[1fr_6rem_7rem] gap-x-2 text-xs text-neutral-500">
                  <span>{t("pages.meal_plan.consume.column.expires_at")}</span>
                  <span>{t("pages.meal_plan.consume.column.current")}</span>
                  <span>{t("pages.meal_plan.consume.column.amount")}</span>
                </div>
                <ul className="mt-1 flex flex-col gap-1">
                  {candidates.map((c, j) => {
                    const i = start + j;
                    return (
                      <li
                        key={c.stockId}
                        data-testid="consume-row"
                        data-expires-at={c.expiresAt}
                        className="grid grid-cols-[1fr_6rem_7rem] items-start gap-x-2 text-sm"
                      >
                        <span className="pt-2">
                          {c.expiresAt ? formatDisplayDate(c.expiresAt) : "—"}
                        </span>
                        <span className="pt-2">
                          {formatQuantity(c.currentAmount, line.unit, measuringUnits)}
                        </span>
                        <div>
                          <Input
                            inputMode="decimal"
                            value={rows[i]?.amount ?? ""}
                            onChange={(ev) => setAmount(i, ev.target.value)}
                            aria-label={t("pages.meal_plan.consume.column.amount")}
                            aria-invalid={!!errors[i]}
                          />
                          <FieldError messageKey={errors[i]} />
                        </div>
                      </li>
                    );
                  })}
                </ul>
              </section>
            );
          })}
          {(plan.missing ?? []).length > 0 && (
            <section aria-labelledby="consume-missing-title" data-testid="consume-missing">
              <h3 id="consume-missing-title" className="text-sm font-semibold text-neutral-700">
                {t("pages.meal_plan.consume.missing_title")}
              </h3>
              <ul className="mt-1 flex flex-wrap gap-x-3 text-sm text-neutral-600">
                {(plan.missing ?? []).map((name) => (
                  <li key={name}>{name}</li>
                ))}
              </ul>
            </section>
          )}
        </div>
      </div>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose}>
          {t("pages.meal_plan.consume.skip")}
        </Button>
        <Button type="button" onClick={submit} disabled={pending}>
          {t("pages.meal_plan.consume.submit")}
        </Button>
      </DialogFooter>
    </>
  );
}
