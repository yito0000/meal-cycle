import { useTranslation } from "react-i18next";

export function MealPlanPage() {
  const { t } = useTranslation();
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold">{t("pages.meal_plan.title")}</h1>
      <p className="mt-2 text-neutral-500">{t("pages.meal_plan.placeholder")}</p>
    </div>
  );
}
