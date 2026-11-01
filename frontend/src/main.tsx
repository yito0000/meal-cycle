import { focusManager, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "@tanstack/react-router";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { router } from "./router";
import "./i18n";
import "./index.css";

// 既定は visibilitychange だけを見るので、別ウィンドウ (AI クライアント) から戻っただけでは取り直さない (add-mcp-server design.md D7)。
// 引数なしで呼ぶと、focus の状態を固定せずに購読側へ知らせる。
focusManager.setEventListener((handleFocus) => {
  const listener = () => handleFocus();
  window.addEventListener("focus", listener, false);
  window.addEventListener("visibilitychange", listener, false);
  return () => {
    window.removeEventListener("focus", listener);
    window.removeEventListener("visibilitychange", listener);
  };
});

const queryClient = new QueryClient();

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("Root element #root not found");
}

createRoot(rootElement).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>,
);
