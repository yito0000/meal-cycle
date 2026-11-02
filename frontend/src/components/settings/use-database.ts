import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { DatabaseService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";

const PENDING_KEY = ["database", "pending"] as const;
const BACKUPS_KEY = ["database", "backups"] as const;

// S09 を開くたびに取り直し、予約中の表示を最新にする (add-settings-database-file design.md D6)。
export function usePendingRestore() {
  return useQuery({
    queryKey: PENDING_KEY,
    queryFn: () => DatabaseService.Pending(),
    staleTime: 0,
  });
}

export function useBackups(enabled: boolean) {
  return useQuery({
    queryKey: BACKUPS_KEY,
    queryFn: async () => (await DatabaseService.ListBackups()) ?? [],
    enabled,
    staleTime: 0,
  });
}

export function useExportDatabase() {
  return useMutation({
    mutationFn: (dialogTitle: string) => DatabaseService.Export(dialogTitle),
  });
}

function useInvalidatePending() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: PENDING_KEY });
}

export function useReserveFromBackup() {
  const invalidate = useInvalidatePending();
  return useMutation({
    mutationFn: (name: string) => DatabaseService.ReserveFromBackup(name),
    onSuccess: invalidate,
  });
}

export function useReserveFromFile() {
  const invalidate = useInvalidatePending();
  return useMutation({
    mutationFn: (dialogTitle: string) => DatabaseService.ReserveFromFile(dialogTitle),
    onSuccess: invalidate,
  });
}

export function useCancelPending() {
  const invalidate = useInvalidatePending();
  return useMutation({
    mutationFn: () => DatabaseService.CancelPending(),
    onSuccess: invalidate,
  });
}

// Go のエラー文字列の部分一致で i18n key を引く。Wails は型付きエラーを運ばないため。
export function databaseErrorKey(err: unknown): string {
  const msg = err instanceof Error ? (err.message ?? "") : String(err ?? "");
  if (msg.includes("restore source is not a meal-cycle database"))
    return "pages.settings.database.errors.not_database";
  if (msg.includes("restore source is newer than this app"))
    return "pages.settings.database.errors.too_new";
  if (msg.includes("backup not found")) return "pages.settings.database.errors.backup_not_found";
  return "pages.settings.database.errors.unknown";
}
