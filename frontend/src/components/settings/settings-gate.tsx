import { createContext, type ReactNode, useContext, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import i18n from "@/i18n";
import { type SettingsValues, useSettings } from "./use-settings";

const SettingsContext = createContext<SettingsValues | null>(null);

// 起動時に別の言語の文言を一瞬でも挟まないよう、保存した表示言語を当て終えるまで子を描かない (design.md D8)。
export function SettingsGate({ children }: { children: ReactNode }) {
  const { t } = useTranslation();
  const { data, isError, refetch, isFetching } = useSettings();
  const [ready, setReady] = useState(false);
  const language = data?.language;

  useEffect(() => {
    if (ready || language === undefined) return;
    let cancelled = false;
    void i18n.changeLanguage(language).then(() => {
      if (!cancelled) setReady(true);
    });
    return () => {
      cancelled = true;
    };
  }, [language, ready]);

  if (isError && !data) {
    return (
      <main className="flex min-h-screen w-full items-start justify-center bg-neutral-50 p-8">
        <div
          role="alert"
          className="flex w-full max-w-xl flex-col gap-4 rounded-lg border border-red-200 bg-white p-6"
        >
          <p className="text-sm text-red-800">{t("settings_gate.load_error")}</p>
          <div>
            <Button type="button" onClick={() => void refetch()} disabled={isFetching}>
              {t("settings_gate.retry")}
            </Button>
          </div>
        </div>
      </main>
    );
  }
  if (!data || !ready) {
    return <div className="min-h-screen w-full bg-neutral-50" />;
  }
  return <SettingsContext.Provider value={data}>{children}</SettingsContext.Provider>;
}

// SettingsGate の内側でだけ呼ぶ。Gate が取得前に子を描かないので undefined を返さない。
export function useSettingsValues(): SettingsValues {
  const values = useContext(SettingsContext);
  if (!values) {
    throw new Error("useSettingsValues must be used inside SettingsGate");
  }
  return values;
}
