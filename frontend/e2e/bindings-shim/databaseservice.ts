import type * as settings$0 from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/settings/models.js";
import { call } from "./call.js";

export function CancelPending() {
  return call<void>("DatabaseService", "CancelPending");
}

export function Export(dialogTitle: string) {
  return call<settings$0.DatabaseExportResult>("DatabaseService", "Export", dialogTitle);
}

export function ListBackups() {
  return call<settings$0.BackupFile[]>("DatabaseService", "ListBackups");
}

export function Pending() {
  return call<settings$0.PendingRestore>("DatabaseService", "Pending");
}

export function Quit() {
  return call<void>("DatabaseService", "Quit");
}

export function ReserveFromBackup(name: string) {
  return call<settings$0.PendingRestore>("DatabaseService", "ReserveFromBackup", name);
}

export function ReserveFromFile(dialogTitle: string) {
  return call<settings$0.ReserveResult>("DatabaseService", "ReserveFromFile", dialogTitle);
}
