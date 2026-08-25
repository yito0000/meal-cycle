import { useTranslation } from "react-i18next";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { RECIPE_SORT_KEYS, type RecipeSortKey } from "./recipe-sort";

type Props = {
  value: RecipeSortKey;
  onChange: (value: RecipeSortKey) => void;
};

function sortLabelKey(key: RecipeSortKey): string {
  switch (key) {
    case "updated-desc":
      return "pages.recipes.sort.updated_desc";
    case "name-asc":
      return "pages.recipes.sort.name_asc";
  }
}

export function RecipeSortSelect({ value, onChange }: Props) {
  const { t } = useTranslation();
  return (
    <div className="flex items-center gap-2 text-sm text-neutral-700">
      <span id="recipe-sort-label">{t("pages.recipes.sort.label")}</span>
      <Select value={value} onValueChange={(v) => onChange(v as RecipeSortKey)}>
        <SelectTrigger aria-labelledby="recipe-sort-label" className="w-40">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {RECIPE_SORT_KEYS.map((k) => (
            <SelectItem key={k} value={k}>
              {t(sortLabelKey(k))}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
