import { useTranslation } from "react-i18next";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { FRIDGE_SORT_KEYS, type FridgeSortKey } from "./fridge-sort";

type Props = {
  value: FridgeSortKey;
  onChange: (value: FridgeSortKey) => void;
};

function sortLabelKey(key: FridgeSortKey): string {
  switch (key) {
    case "expiry-asc":
      return "pages.fridge.sort.expiry_asc";
    case "name-asc":
      return "pages.fridge.sort.name_asc";
    case "added-desc":
      return "pages.fridge.sort.added_desc";
  }
}

export function FridgeSortSelect({ value, onChange }: Props) {
  const { t } = useTranslation();
  return (
    <div className="flex items-center gap-2 text-sm text-neutral-700">
      <span id="fridge-sort-label">{t("pages.fridge.sort.label")}</span>
      <Select value={value} onValueChange={(v) => onChange(v as FridgeSortKey)}>
        <SelectTrigger aria-labelledby="fridge-sort-label" className="w-44">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {FRIDGE_SORT_KEYS.map((k) => (
            <SelectItem key={k} value={k}>
              {t(sortLabelKey(k))}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
