import { useTranslation } from "react-i18next";
import { Input } from "@/components/ui/input";

type Props = {
  value: string;
  onChange: (value: string) => void;
};

export function FridgeSearchInput({ value, onChange }: Props) {
  const { t } = useTranslation();
  return (
    <Input
      type="search"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={t("pages.fridge.search.placeholder")}
      aria-label={t("pages.fridge.search.placeholder")}
      className="max-w-sm"
    />
  );
}
