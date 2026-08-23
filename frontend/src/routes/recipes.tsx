import { useTranslation } from "react-i18next";

export function RecipesPage() {
  const { t } = useTranslation();
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold">{t("pages.recipes.title")}</h1>
      <p className="mt-2 text-neutral-500">{t("pages.recipes.placeholder")}</p>
    </div>
  );
}
