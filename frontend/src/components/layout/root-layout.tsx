import { Outlet } from "@tanstack/react-router";
import { SettingsGate } from "@/components/settings/settings-gate";
import { Toaster } from "@/components/ui/sonner";
import { Sidebar } from "./sidebar";
import { StartupErrorScreen } from "./startup-error-screen";
import { useHealth } from "./use-health";

export function RootLayout() {
  const { data: health, isPending } = useHealth();

  // 起動に失敗したときは HealthService 以外が登録されていないので、ページを描く前に判定を待つ (design.md D2)。
  if (isPending) {
    return <div className="min-h-screen bg-neutral-50" />;
  }
  // SettingsService も登録されていないので、エラー画面は SettingsGate の外で既定の日本語のまま出す。
  if (health?.startupError) {
    return <StartupErrorScreen health={health} />;
  }

  return (
    <>
      <SettingsGate>
        <div className="flex min-h-screen bg-neutral-50">
          <Sidebar />
          <main className="flex-1 overflow-auto">
            <Outlet />
          </main>
        </div>
      </SettingsGate>
      <Toaster position="bottom-center" />
    </>
  );
}
