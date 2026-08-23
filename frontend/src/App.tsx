import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Events } from "@wailsio/runtime";
import { HealthService } from "../bindings/github.com/yito0000/meal-cycle";
import type { Health } from "../bindings/github.com/yito0000/meal-cycle/models";

export function App() {
  const { t } = useTranslation();
  const [health, setHealth] = useState<Health | null>(null);
  const [time, setTime] = useState("");

  useEffect(() => {
    HealthService.Ping().then(setHealth).catch(console.error);
  }, []);

  useEffect(() => {
    const cancel = Events.On("time", (event) => {
      setTime(String(event.data));
    });
    return () => {
      if (typeof cancel === "function") cancel();
    };
  }, []);

  return (
    <main className="mx-auto flex min-h-screen max-w-2xl flex-col gap-8 p-8">
      <header>
        <h1 className="text-3xl font-bold">meal-cycle</h1>
        <p className="text-sm text-neutral-500">{t("app.tagline")}</p>
      </header>

      <section className="rounded border border-neutral-200 p-4 text-sm">
        <p className="font-bold">{t("health.title")}</p>
        {health === null ? (
          <p className="text-neutral-500">{t("health.loading")}</p>
        ) : health.dbConnected ? (
          <p className="text-green-700">{t("health.db_ok")}</p>
        ) : (
          <p className="text-red-700">
            {t("health.db_error", { error: health.dbError ?? "" })}
          </p>
        )}
      </section>

      <footer className="mt-auto text-xs text-neutral-400">
        {time || t("footer.waiting_time")}
      </footer>
    </main>
  );
}
