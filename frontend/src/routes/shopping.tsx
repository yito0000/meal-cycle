import { useTranslation } from "react-i18next";

export function ShoppingPage() {
  const { t } = useTranslation();
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold">{t("pages.shopping.title")}</h1>
      <p className="mt-2 text-neutral-500">{t("pages.shopping.placeholder")}</p>
    </div>
  );
}
