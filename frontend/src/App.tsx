import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Events } from "@wailsio/runtime";
import { GreetService } from "../bindings/changeme";

export function App() {
  const { t } = useTranslation();
  const [name, setName] = useState("");
  const [greeting, setGreeting] = useState("");
  const [time, setTime] = useState("");

  useEffect(() => {
    const cancel = Events.On("time", (event: { data: string }) => {
      setTime(event.data);
    });
    return () => {
      if (typeof cancel === "function") cancel();
    };
  }, []);

  const handleGreet = async () => {
    try {
      const message = await GreetService.Greet(name || "anonymous");
      setGreeting(message);
    } catch (err) {
      console.error(err);
    }
  };

  return (
    <main className="mx-auto flex min-h-screen max-w-2xl flex-col gap-8 p-8">
      <header>
        <h1 className="text-3xl font-bold">meal-cycle</h1>
        <p className="text-sm text-neutral-500">{t("app.tagline")}</p>
      </header>

      <section className="flex flex-col gap-3">
        <label className="flex flex-col gap-1 text-sm">
          {t("greet.name_label")}
          <input
            className="rounded border border-neutral-300 px-3 py-2 text-base"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t("greet.name_placeholder")}
          />
        </label>
        <button
          className="self-start rounded bg-neutral-900 px-4 py-2 text-white hover:bg-neutral-700"
          onClick={handleGreet}
        >
          {t("greet.button")}
        </button>
        {greeting && (
          <p className="rounded bg-neutral-100 p-3 text-sm">{greeting}</p>
        )}
      </section>

      <footer className="mt-auto text-xs text-neutral-400">
        {time || t("footer.waiting_time")}
      </footer>
    </main>
  );
}
