import { spawn } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test as base } from "@playwright/test";
import { REPO_ROOT, SERVER_BIN } from "./global-setup.js";

// cmd/e2e-server が出す起動完了行。文言を変えるときは main.go も一緒に直すこと。
const LISTENING_PREFIX = "E2E_SERVER_LISTENING ";
const STARTUP_TIMEOUT_MS = 15_000;

type TestFixtures = {
  // describe 側で test.use({ seed: true }) と書くとサンプルレシピ入りで起動する。
  // 空状態のシナリオがあるため既定は false (design.md D6)。
  seed: boolean;
  serverURL: string;
  baseURL: string;
};

// サーバと DB はテストごとに作り直す。worker で使い回すとテスト間で状態が
// 漏れ、実行順序に依存する (design.md D6)。起動は 100ms 前後で済む。
export const test = base.extend<TestFixtures>({
  seed: [false, { option: true }],

  serverURL: async ({ seed }, use) => {
    const dir = mkdtempSync(join(tmpdir(), "meal-cycle-e2e-"));
    const args = ["-db", join(dir, "e2e.db"), "-dist", "frontend/dist", "-port", "0"];
    if (seed) {
      args.push("-seed");
    }

    const proc = spawn(SERVER_BIN, args, { cwd: REPO_ROOT });
    try {
      await use(await waitForURL(proc));
    } finally {
      proc.kill();
      rmSync(dir, { recursive: true, force: true });
    }
  },

  baseURL: async ({ serverURL }, use) => {
    await use(serverURL);
  },
});

export { expect } from "@playwright/test";

// waitForURL は stdout の起動完了行から実ポートを読む。固定ポートを使わないのは
// worker 並列と他プロセスの衝突を避けるため (design.md D7)。
function waitForURL(proc: ReturnType<typeof spawn>): Promise<string> {
  return new Promise((resolve, reject) => {
    let buffered = "";
    const timer = setTimeout(() => {
      reject(new Error(`e2e-server did not start in ${STARTUP_TIMEOUT_MS}ms:\n${buffered}`));
    }, STARTUP_TIMEOUT_MS);

    const finish = (err: Error | null, url?: string) => {
      clearTimeout(timer);
      if (err) {
        reject(err);
      } else if (url) {
        resolve(url);
      }
    };

    proc.stdout?.on("data", (chunk: Buffer) => {
      buffered += chunk.toString();
      const line = buffered.split("\n").find((l) => l.startsWith(LISTENING_PREFIX));
      if (line) {
        finish(null, line.slice(LISTENING_PREFIX.length).trim());
      }
    });
    proc.stderr?.on("data", (chunk: Buffer) => {
      buffered += chunk.toString();
    });
    proc.on("error", finish);
    proc.on("exit", (code) => finish(new Error(`e2e-server exited with ${code}:\n${buffered}`)));
  });
}
