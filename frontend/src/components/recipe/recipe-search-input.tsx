import { useTranslation } from "react-i18next";
import { Input } from "@/components/ui/input";

type Props = {
  value: string;
  onChange: (value: string) => void;
};

export function RecipeSearchInput({ value, onChange }: Props) {
  const { t } = useTranslation();
  return (
    <Input
      type="search"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={t("pages.recipes.search.placeholder")}
      aria-label={t("pages.recipes.search.placeholder")}
      className="max-w-sm"
    />
  );
}
