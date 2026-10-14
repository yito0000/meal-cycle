import { useTranslation } from "react-i18next";
import type { Health } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service/models";

const knownStages = ["open", "backup", "migrate"] as const;

export function StartupErrorScreen({ health }: { health: Health }) {
  const { t } = useTranslation();
  const stage = knownStages.find((s) => s === health.startupError) ?? "open";

  return (
    <main className="flex min-h-screen items-start justify-center bg-neutral-50 p-8">
      <div
        role="alert"
        className="flex w-full max-w-3xl flex-col gap-5 rounded-lg border border-red-200 bg-white p-6"
      >
        <div className="flex flex-col gap-1">
          <p className="text-sm text-neutral-500">{t("startup_error.title")}</p>
          <h1 className="text-xl font-bold text-red-800">{t(`startup_error.stage.${stage}`)}</h1>
        </div>

        {health.startupDetail ? (
          <section className="flex flex-col gap-1">
            <h2 className="text-sm font-semibold text-neutral-700">{t("startup_error.detail")}</h2>
            <pre className="overflow-x-auto whitespace-pre-wrap break-all rounded-md bg-neutral-100 p-3 font-mono text-sm text-neutral-900 select-text">
              {health.startupDetail}
            </pre>
          </section>
        ) : null}

        <PathRow label={t("startup_error.db_path")} value={health.dbPath} />
        <PathRow label={t("startup_error.backup_dir")} value={health.backupDir} />

        <p className="text-sm text-neutral-600">{t("startup_error.hint")}</p>
      </div>
    </main>
  );
}

function PathRow({ label, value }: { label: string; value?: string }) {
  if (!value) return null;
  return (
    <section className="flex flex-col gap-1">
      <h2 className="text-sm font-semibold text-neutral-700">{label}</h2>
      <code className="break-all rounded-md bg-neutral-100 p-3 font-mono text-sm text-neutral-900 select-text">
        {value}
      </code>
    </section>
  );
}
