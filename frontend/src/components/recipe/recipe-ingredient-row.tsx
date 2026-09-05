import { X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { RecipeIngredient } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";

type Props = {
  value: RecipeIngredient;
  onChange: (next: RecipeIngredient) => void;
  onRemove: () => void;
  nameError?: string | null;
};

// 材料 1 行の入力コンポーネント。食材コンボボックスは持たず、名前 / 量 / 単位 の
// 3 テキスト input と削除ボタンだけの単純な行 (design.md D9 参照)。
export function RecipeIngredientRow({ value, onChange, onRemove, nameError }: Props) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-1">
      <div className="grid grid-cols-[minmax(0,1fr)_5rem_5rem_2.25rem] items-center gap-2">
        <Input
          type="text"
          value={value.name}
          onChange={(e) => onChange({ ...value, name: e.target.value })}
          placeholder={t("pages.recipes.edit.label.name")}
          aria-label={t("pages.recipes.edit.label.name")}
          aria-invalid={nameError != null}
        />
        <Input
          type="number"
          inputMode="decimal"
          step="any"
          min={0}
          value={Number.isFinite(value.amount) ? value.amount : 0}
          onChange={(e) => {
            const parsed = Number.parseFloat(e.target.value);
            onChange({ ...value, amount: Number.isNaN(parsed) ? 0 : parsed });
          }}
          aria-label={t("pages.recipes.edit.label.amount")}
        />
        <Input
          type="text"
          value={value.unit}
          onChange={(e) => onChange({ ...value, unit: e.target.value })}
          aria-label={t("pages.recipes.edit.label.unit")}
        />
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
