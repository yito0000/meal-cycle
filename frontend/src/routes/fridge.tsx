import { useTranslation } from "react-i18next";

export function FridgePage() {
  const { t } = useTranslation();
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold">{t("pages.fridge.title")}</h1>
      <p className="mt-2 text-neutral-500">{t("pages.fridge.placeholder")}</p>
    </div>
  );
}
