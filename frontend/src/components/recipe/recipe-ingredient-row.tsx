import { X } from "lucide-react";
import { useId } from "react";
import { useTranslation } from "react-i18next";
import { IngredientNameInput } from "@/components/ingredient/ingredient-name-input";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { NameSuggestion } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import type { IngredientFormRow } from "./use-recipe-form";

type Props = {
  value: IngredientFormRow;
  onChange: (next: IngredientFormRow) => void;
  onRemove: () => void;
  nameError?: string | null;
  nameSuggestions: NameSuggestion[];
  unitSuggestions: string[];
};

// 名前 / 量 / 単位 + 削除ボタン。名前は S06 と同じ候補補完を持つ (add-ingredient-name-matching design.md D4)。
export function RecipeIngredientRow({
  value,
  onChange,
  onRemove,
  nameError,
  nameSuggestions,
  unitSuggestions,
}: Props) {
  const { t } = useTranslation();
  const unitListId = useId();
  return (
    <div className="flex flex-col gap-1">
      <div className="grid grid-cols-[minmax(0,1fr)_5rem_5rem_2.25rem] items-center gap-2">
        <IngredientNameInput
          value={value.name}
          onChange={(name) => onChange({ ...value, name })}
          onSelect={(s) =>
            onChange({
              ...value,
              name: s.name,
              unit: value.unit.trim() === "" ? s.unit : value.unit,
            })
          }
          suggestions={nameSuggestions}
          placeholder={t("pages.recipes.edit.label.name")}
          aria-label={t("pages.recipes.edit.label.name")}
          aria-invalid={nameError != null}
        />
        <Input
          type="number"
          inputMode="decimal"
          step="any"
          min={0}
          value={value.amount}
          onChange={(e) => onChange({ ...value, amount: e.target.value })}
          aria-label={t("pages.recipes.edit.label.amount")}
        />
        <div>
          <Input
            type="text"
            list={unitListId}
            value={value.unit}
            onChange={(e) => onChange({ ...value, unit: e.target.value })}
            aria-label={t("pages.recipes.edit.label.unit")}
          />
          <datalist id={unitListId}>
            {unitSuggestions.map((u) => (
              <option key={u} value={u} />
            ))}
          </datalist>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          onClick={onRemove}
          aria-label={t("pages.recipes.edit.remove_ingredient")}
        >
          <X className="h-4 w-4" />
        </Button>
      </div>
      {nameError && <p className="text-xs text-red-700">{t(nameError)}</p>}
    </div>
  );
}
