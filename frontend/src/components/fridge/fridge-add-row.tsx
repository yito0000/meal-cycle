import * as React from "react";
import { useTranslation } from "react-i18next";
import { IngredientNameInput } from "@/components/ingredient/ingredient-name-input";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { isImeComposing } from "@/lib/ime";
import type { NameSuggestion } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { stockErrorOf } from "./error-messages";
import { FieldError } from "./field-errors";
import {
  emptyStockForm,
  type StockField,
  type StockFormErrors,
  type StockFormValues,
  toStockInput,
  validateStockForm,
} from "./stock-form";
import { useAddStock } from "./use-fridge";

type Props = {
  nameSuggestions: NameSuggestion[];
  unitSuggestions: string[];
};

// S06 追加行。S05 の一覧上部に常駐し、追加後は入力をクリアして連続追加できる。
export function FridgeAddRow({ nameSuggestions, unitSuggestions }: Props) {
  const { t } = useTranslation();
  const unitListId = React.useId();
  const nameRef = React.useRef<HTMLInputElement>(null);
  const [values, setValues] = React.useState<StockFormValues>(emptyStockForm);
  const [errors, setErrors] = React.useState<StockFormErrors>({});
  const [formError, setFormError] = React.useState<string>();
  const add = useAddStock();

  const set = (field: StockField) => (value: string) => {
    setValues((v) => ({ ...v, [field]: value }));
    setErrors((e) => ({ ...e, [field]: undefined }));
  };

  const submit = () => {
    if (add.isPending) return;
    const found = validateStockForm(values);
    setErrors(found);
    setFormError(undefined);
    if (Object.keys(found).length > 0) return;
    add.mutate(toStockInput(values), {
      onSuccess: () => {
        setValues(emptyStockForm);
        nameRef.current?.focus();
      },
      onError: (err) => {
        const { key, field } = stockErrorOf(err);
        if (field) setErrors({ [field]: key });
        else setFormError(key);
      },
    });
  };

  // 食材名欄の Enter は候補確定のためにあり、送信には使わない (spec: インライン追加行)。
  const onKeyDown = (e: React.KeyboardEvent<HTMLFieldSetElement>) => {
    if (e.key !== "Enter" || e.defaultPrevented || isImeComposing(e)) return;
    if (e.target === nameRef.current) return;
    e.preventDefault();
    submit();
  };

  return (
    // 各入力欄の Enter をここで集約して送信する。
    <fieldset
      aria-label={t("pages.fridge.add.label")}
      className="min-w-0 rounded-md border border-neutral-200 bg-white p-3"
      onKeyDown={onKeyDown}
    >
      <div className="flex flex-wrap items-start gap-2">
        <span className="pt-2 text-sm font-medium text-neutral-700">
          {t("pages.fridge.add.label")}
        </span>
        <div className="min-w-48 flex-1">
          <IngredientNameInput
            ref={nameRef}
            value={values.name}
            onChange={set("name")}
            onSelect={(s) => {
              setValues((v) => ({
                ...v,
                name: s.name,
                unit: v.unit.trim() === "" ? s.unit : v.unit,
              }));
              setErrors((e) => ({ ...e, name: undefined }));
            }}
            suggestions={nameSuggestions}
            placeholder={t("pages.fridge.add.name_placeholder")}
            aria-label={t("pages.fridge.field.name")}
            aria-invalid={!!errors.name}
          />
          <FieldError messageKey={errors.name} />
        </div>
        <div className="w-24">
          <Input
            type="number"
            inputMode="decimal"
            step="any"
            min="0"
            value={values.amount}
            onChange={(e) => set("amount")(e.target.value)}
            placeholder={t("pages.fridge.add.amount_placeholder")}
            aria-label={t("pages.fridge.field.amount")}
            aria-invalid={!!errors.amount}
          />
          <FieldError messageKey={errors.amount} />
        </div>
        <div className="w-24">
          <Input
            list={unitListId}
            value={values.unit}
            onChange={(e) => set("unit")(e.target.value)}
            placeholder={t("pages.fridge.add.unit_placeholder")}
            aria-label={t("pages.fridge.field.unit")}
            aria-invalid={!!errors.unit}
          />
          <datalist id={unitListId}>
            {unitSuggestions.map((u) => (
              <option key={u} value={u} />
            ))}
          </datalist>
          <FieldError messageKey={errors.unit} />
        </div>
        <div className="w-44">
          <Input
            type="date"
            value={values.expiresAt}
            onChange={(e) => set("expiresAt")(e.target.value)}
            title={t("pages.fridge.add.expires_placeholder")}
            aria-label={t("pages.fridge.field.expires_at")}
            aria-invalid={!!errors.expiresAt}
          />
          <FieldError messageKey={errors.expiresAt} />
        </div>
        <Button type="button" onClick={submit} disabled={add.isPending}>
          {t("pages.fridge.add.submit")}
        </Button>
      </div>
      <FieldError messageKey={formError} />
    </fieldset>
  );
}
