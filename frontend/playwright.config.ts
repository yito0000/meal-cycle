import { defineConfig } from "@playwright/test";

// ブラウザは headless Chromium 1 種類に絞る。Wails の webview は WebKit だが、
// E2E で検証したいのはアプリのロジックであって描画エンジン差ではない。
// 実機の描画確認は Windows ビルドの目視に任せる (design.md Risks)。
export default defineConfig({
  testDir: "./e2e",
  globalSetup: "./e2e/global-setup.ts",

  // spec 内は直列。1 spec = 1 サーバ = 1 DB で、テストが状態を共有しないため
  // 順序依存を持ち込まない (design.md D6)。spec 間は worker 並列で走る。
  fullyParallel: false,

  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? "list" : [["list"], ["html", { open: "never" }]],

  use: {
    trace: "retain-on-failure",
  },

  projects: [{ name: "chromium", use: { browserName: "chromium" } }],
});
