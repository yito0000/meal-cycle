import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { formatTimestampDate } from "@/lib/format-date";
import type { BackupFile } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/settings/models";
import {
  databaseErrorKey,
  useBackups,
  useReserveFromBackup,
  useReserveFromFile,
} from "./use-database";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

function timeOf(rfc3339: string): string {
  const d = new Date(rfc3339);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${formatTimestampDate(rfc3339)} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

// 確認の前に選んだもの。控えは名前で、ファイルは選択ダイアログを開くまで中身が決まらないので印だけを持つ。
type Choice = { kind: "backup"; backup: BackupFile } | { kind: "file" };

export function RestoreDialog({ open, onOpenChange }: Props) {
  const { t } = useTranslation();
  const backups = useBackups(open);
  const fromBackup = useReserveFromBackup();
  const fromFile = useReserveFromFile();
  const [choice, setChoice] = useState<Choice | null>(null);
  const [error, setError] = useState<string>();
  const pending = fromBackup.isPending || fromFile.isPending;

  const close = () => {
    setChoice(null);
    setError(undefined);
    onOpenChange(false);
  };

  const onReserved = (name: string) => {
    toast(t("pages.settings.database.reserved", { name }));
    close();
  };

  const confirm = () => {
    if (!choice) return;
    setError(undefined);
    if (choice.kind === "backup") {
      fromBackup.mutate(choice.backup.name, {
        onSuccess: (p) => onReserved(p.name),
        onError: (err) => setError(databaseErrorKey(err)),
      });
      return;
    }
    fromFile.mutate(t("pages.settings.database.open_dialog_title"), {
      onSuccess: (r) => {
        if (r.cancelled) {
          setChoice(null);
          return;
        }
        onReserved(r.pending.name);
      },
      onError: (err) => setError(databaseErrorKey(err)),
    });
  };

  const list = backups.data ?? [];
  return (
    <Dialog open={open} onOpenChange={(next) => (next ? onOpenChange(true) : close())}>
      <DialogContent className="flex max-h-[85vh] max-w-xl flex-col">
        {choice === null ? (
          <>
            <DialogHeader>
              <DialogTitle>{t("pages.settings.database.restore_title")}</DialogTitle>
              <DialogDescription>
                {t("pages.settings.database.restore_description")}
              </DialogDescription>
            </DialogHeader>
            <div className="-mx-1 min-h-0 flex-1 overflow-y-auto px-1">
              {backups.isLoading && (
                <p className="text-sm text-neutral-500">{t("pages.settings.database.loading")}</p>
              )}
              {backups.data && list.length === 0 && (
                <p className="text-sm text-neutral-500">
                  {t("pages.settings.database.no_backups")}
                </p>
              )}
              <ul className="flex flex-col gap-2">
                {list.map((b) => (
                  <li key={b.name}>
                    <button
                      type="button"
                      data-testid="backup-item"
                      className="flex w-full items-center justify-between gap-3 rounded-md border border-neutral-200 px-3 py-2 text-left text-sm hover:bg-neutral-50"
                      onClick={() => setChoice({ kind: "backup", backup: b })}
                    >
                      <span className="tabular-nums">{timeOf(b.createdAt)}</span>
                      <span className="text-neutral-600">
                        {b.beforeRestore
                          ? t("pages.settings.database.backup_before_restore")
                          : t("pages.settings.database.backup_before_migrate", {
                              version: b.version,
                            })}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            </div>
            <DialogFooter className="gap-2">
              <Button type="button" variant="outline" onClick={close}>
                {t("pages.settings.database.cancel")}
              </Button>
              <Button type="button" onClick={() => setChoice({ kind: "file" })}>
                {t("pages.settings.database.pick_file")}
              </Button>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>{t("pages.settings.database.confirm_title")}</DialogTitle>
              <DialogDescription>{t("pages.settings.database.confirm_body")}</DialogDescription>
            </DialogHeader>
            {choice.kind === "backup" && (
              <p className="text-sm text-neutral-700">{timeOf(choice.backup.createdAt)}</p>
            )}
            {error && (
              <p role="alert" className="text-sm text-red-700">
                {t(error)}
              </p>
            )}
            <DialogFooter className="gap-2">
              <Button
                type="button"
                variant="outline"
                disabled={pending}
                onClick={() => {
                  setChoice(null);
                  setError(undefined);
                }}
              >
                {t("pages.settings.database.back")}
              </Button>
              <Button type="button" disabled={pending} onClick={confirm}>
                {choice.kind === "file"
                  ? t("pages.settings.database.confirm_pick_file")
                  : t("pages.settings.database.confirm")}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
