import path from "node:path";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// 日付ヘルパーはローカル TZ で日付を組み立てるため、実行環境に依らず利用者と同じ TZ に揃える (design.md D4)。
process.env.TZ = "Asia/Tokyo";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      // vite.config.ts と同じ alias。変えるときは両方を直す。
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
  test: {
    include: ["src/**/*.test.{ts,tsx}"],
    environment: "happy-dom",
  },
});
