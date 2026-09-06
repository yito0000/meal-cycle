import path from "node:path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import wails from "@wailsio/runtime/plugins/vite";
import { defineConfig } from "vite";
import e2eBindings from "./vite-plugin-e2e-bindings";

export default defineConfig(({ mode }) => ({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
  // E2E ビルドのときだけ bindings を fetch 版 shim に差し替える (design.md D2)。
  plugins: [
    react(),
    tailwindcss(),
    wails("./bindings"),
    ...(mode === "e2e" ? [e2eBindings("./e2e/bindings-shim")] : []),
  ],
}));
