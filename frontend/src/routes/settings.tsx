import { useTranslation } from "react-i18next";
import { SettingsForm } from "../components/settings/settings-form";
import { useSettingsValues } from "../components/settings/settings-gate";

export function SettingsPage() {
  const { t } = useTranslation();
  const saved = useSettingsValues();

  return (
    <div className="flex h-full flex-col gap-4 p-6">
      <h1 className="text-2xl font-bold">{t("pages.settings.title")}</h1>
      <SettingsForm saved={saved} />
    </div>
  );
}
