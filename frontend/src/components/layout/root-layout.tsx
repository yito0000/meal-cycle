import { Outlet } from "@tanstack/react-router";
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
  if (health?.startupError) {
    return <StartupErrorScreen health={health} />;
  }

  return (
    <div className="flex min-h-screen bg-neutral-50">
      <Sidebar />
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
      <Toaster position="bottom-center" />
    </div>
  );
}
