import { execFileSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

// URL の pathname をそのまま使うと Windows で "/D:/..." になり、パスが壊れる。
export const REPO_ROOT = fileURLToPath(new URL("../..", import.meta.url));

// test-results/ は Playwright が実行前に消すので、バイナリは node_modules 配下に置く。
const BIN_DIR = join(REPO_ROOT, "frontend", "node_modules", ".cache");
// Windows は拡張子が無いと spawn で起動できない。
export const SERVER_BIN = join(
  BIN_DIR,
  process.platform === "win32" ? "e2e-server.exe" : "e2e-server",
);

// Go 側サーバをテスト開始前に 1 度だけビルドする。worker ごとに go run すると
// ビルドが並列に走って起動が遅くなるため、バイナリを先に作って使い回す。
export default function globalSetup() {
  mkdirSync(BIN_DIR, { recursive: true });
  execFileSync("go", ["build", "-o", SERVER_BIN, "./cmd/e2e-server"], {
    cwd: REPO_ROOT,
    stdio: "inherit",
  });
}
