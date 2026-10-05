import * as React from "react";
import { useTranslation } from "react-i18next";
import { IngredientNameInput } from "@/components/ingredient/ingredient-name-input";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { TableCell, TableRow } from "@/components/ui/table";
import { isImeComposing } from "@/lib/ime";
import type {
  NameSuggestion,
  StockItem,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { stockErrorOf } from "./error-messages";
import { FieldError } from "./field-errors";
import {
  fromStockItem,
  type StockField,
  type StockFormErrors,
  type StockFormValues,
  toStockInput,
  validateStockForm,
} from "./stock-form";
import { useUpdateStock } from "./use-fridge";

type Props = {
  item: StockItem;
  nameSuggestions: NameSuggestion[];
  onDone: () => void;
};

// S06 行内編集。ドロワー / モーダルは開かず、行のセルを入力欄に差し替える。
export function FridgeEditRow({ item, nameSuggestions, onDone }: Props) {
  const { t } = useTranslation();
  const [values, setValues] = React.useState<StockFormValues>(() => fromStockItem(item));
  const [errors, setErrors] = React.useState<StockFormErrors>({});
  const [formError, setFormError] = React.useState<string>();
  const update = useUpdateStock();

  const set = (field: StockField) => (value: string) => {
    setValues((v) => ({ ...v, [field]: value }));
    setErrors((e) => ({ ...e, [field]: undefined }));
  };

  const save = () => {
    if (update.isPending) return;
    const found = validateStockForm(values);
    setErrors(found);
    setFormError(undefined);
    if (Object.keys(found).length > 0) return;
    update.mutate(
      { id: item.id, input: toStockInput(values) },
      {
        onSuccess: onDone,
        onError: (err) => {
          const { key, field } = stockErrorOf(err);
          if (field) setErrors({ [field]: key });
          else setFormError(key);
        },
      },
    );
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLTableRowElement>) => {
    if (e.defaultPrevented || isImeComposing(e)) return;
    if (e.key === "Enter") {
      e.preventDefault();
      save();
    } else if (e.key === "Escape") {
      e.preventDefault();
      onDone();
    }
  };

  return (
    <TableRow
      data-stock-id={item.id}
      data-editing="true"
      className="bg-blue-50/40"
      onKeyDown={onKeyDown}
    >
      <TableCell className="align-top">
        <IngredientNameInput
          autoFocus
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
          aria-label={t("pages.fridge.field.name")}
          aria-invalid={!!errors.name}
        />
        <FieldError messageKey={errors.name} />
        <FieldError messageKey={formError} />
      </TableCell>
      <TableCell className="align-top">
        <div className="flex gap-1">
          <Input
            type="number"
            inputMode="decimal"
            step="any"
            min="0"
            className="w-20"
            value={values.amount}
            onChange={(e) => set("amount")(e.target.value)}
            aria-label={t("pages.fridge.field.amount")}
            aria-invalid={!!errors.amount}
          />
          <Input
            className="w-16"
            value={values.unit}
            onChange={(e) => set("unit")(e.target.value)}
            aria-label={t("pages.fridge.field.unit")}
            aria-invalid={!!errors.unit}
          />
        </div>
        <FieldError messageKey={errors.amount} />
        <FieldError messageKey={errors.unit} />
      </TableCell>
      <TableCell className="align-top">
        <Input
          type="date"
          className="w-40"
          value={values.expiresAt}
          onChange={(e) => set("expiresAt")(e.target.value)}
          aria-label={t("pages.fridge.field.expires_at")}
          aria-invalid={!!errors.expiresAt}
        />
        <FieldError messageKey={errors.expiresAt} />
      </TableCell>
      <TableCell />
      <TableCell className="align-top">
        <div className="flex gap-2">
          <Button type="button" size="sm" onClick={save} disabled={update.isPending}>
            {t("pages.fridge.row.save")}
          </Button>
          <Button type="button" variant="outline" size="sm" onClick={onDone}>
            {t("pages.fridge.row.cancel")}
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}
