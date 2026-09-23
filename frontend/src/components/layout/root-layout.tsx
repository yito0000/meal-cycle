import { Outlet } from "@tanstack/react-router";
import { Toaster } from "@/components/ui/sonner";
import { Sidebar } from "./sidebar";

export function RootLayout() {
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
