import { useTranslation } from "react-i18next";
import { SettingsForm } from "../components/settings/settings-form";
import { useSettingsValues } from "../components/settings/settings-gate";
import { useSettingsConstraints } from "../components/settings/use-settings";
import { Button } from "../components/ui/button";

export function SettingsPage() {
  const { t } = useTranslation();
  const saved = useSettingsValues();
  const { data: constraints, isError, refetch, isFetching } = useSettingsConstraints();

  return (
    <div className="flex h-full flex-col gap-4 p-6">
      <h1 className="text-2xl font-bold">{t("pages.settings.title")}</h1>
      {constraints ? (
        <SettingsForm saved={saved} constraints={constraints} />
      ) : (
        isError && (
          <div role="alert" className="flex flex-col gap-3">
            <p className="text-sm text-red-700">{t("settings_gate.load_error")}</p>
            <div>
              <Button type="button" onClick={() => void refetch()} disabled={isFetching}>
                {t("settings_gate.retry")}
              </Button>
            </div>
          </div>
        )
      )}
    </div>
  );
}
