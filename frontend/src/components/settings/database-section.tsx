import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { DatabaseService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { RestoreDialog } from "./restore-dialog";
import { useCancelPending, useExportDatabase, usePendingRestore } from "./use-database";

// S09 の最後の節。設定の「保存」の対象には含めず、各操作はその場で行う (add-settings-database-file proposal)。
export function DatabaseSection() {
  const { t } = useTranslation();
  const pending = usePendingRestore();
  const exportDb = useExportDatabase();
  const cancel = useCancelPending();
  const [restoreOpen, setRestoreOpen] = useState(false);

  const onExport = () => {
    exportDb.mutate(t("pages.settings.database.save_dialog_title"), {
      onSuccess: (r) => {
        if (r.cancelled) return;
        toast(t("pages.settings.database.exported"), { description: r.path });
      },
      onError: (err) =>
        toast.error(t("pages.settings.database.export_failed"), {
          description: err instanceof Error ? err.message : String(err ?? ""),
        }),
    });
  };

  return (
    <section
      aria-labelledby="settings-database-title"
      className="flex max-w-3xl flex-col gap-4 rounded-lg border border-neutral-200 bg-white p-5"
    >
      <h2 id="settings-database-title" className="text-base font-semibold text-neutral-900">
        {t("pages.settings.section.database")}
      </h2>

      {pending.data?.pending && (
        <div
          data-testid="restore-pending"
          className="flex flex-col gap-3 rounded-md border border-amber-200 bg-amber-50 p-3"
        >
          <p className="text-sm text-amber-900">
            {t("pages.settings.database.pending", { name: pending.data.name })}
          </p>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={cancel.isPending}
              onClick={() =>
                cancel.mutate(undefined, {
                  onSuccess: () => toast(t("pages.settings.database.cancelled")),
                })
              }
            >
              {t("pages.settings.database.cancel_pending")}
            </Button>
            <Button type="button" size="sm" onClick={() => void DatabaseService.Quit()}>
              {t("pages.settings.database.quit")}
            </Button>
          </div>
        </div>
      )}

      <div className="flex flex-col gap-2">
        <div>
          <Button type="button" variant="outline" disabled={exportDb.isPending} onClick={onExport}>
            {t("pages.settings.database.export")}
          </Button>
        </div>
        <p className="text-xs text-neutral-500">{t("pages.settings.database.export_hint")}</p>
      </div>
      <div className="flex flex-col gap-2">
        <div>
          <Button type="button" variant="outline" onClick={() => setRestoreOpen(true)}>
            {t("pages.settings.database.restore")}
          </Button>
        </div>
        <p className="text-xs text-neutral-500">{t("pages.settings.database.restore_hint")}</p>
      </div>

      <RestoreDialog open={restoreOpen} onOpenChange={setRestoreOpen} />
    </section>
  );
}
