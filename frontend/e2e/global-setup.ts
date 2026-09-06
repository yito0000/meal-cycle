import { execFileSync } from "node:child_process";
import { mkdirSync } from "node:fs";

export const REPO_ROOT = new URL("../../", import.meta.url).pathname.replace(/\/$/, "");

// test-results/ は Playwright が実行前に消すので、バイナリは node_modules 配下に置く。
const BIN_DIR = `${REPO_ROOT}/frontend/node_modules/.cache`;
export const SERVER_BIN = `${BIN_DIR}/e2e-server`;

// Go 側サーバをテスト開始前に 1 度だけビルドする。worker ごとに go run すると
// ビルドが並列に走って起動が遅くなるため、バイナリを先に作って使い回す。
export default function globalSetup() {
  mkdirSync(BIN_DIR, { recursive: true });
  execFileSync("go", ["build", "-o", SERVER_BIN, "./cmd/e2e-server"], {
    cwd: REPO_ROOT,
    stdio: "inherit",
  });
}
