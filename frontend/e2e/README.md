# E2E テスト

Wails の IPC 層だけを HTTP に差し替え、React と Go サービスと SQLite は本物のまま通す。
webview (WebKit2GTK) は外部ドライバから掴めないため、ブラウザは headless Chromium を使う。
設計の経緯は [../../openspec/changes/archive/2026-09-07-add-e2e-harness/design.md](../../openspec/changes/archive/2026-09-07-add-e2e-harness/design.md) にある。

## 実行

```bash
task test:e2e
```

型チェック、`vite build --mode e2e`、`playwright test` の順に走る。
Playwright だけを回すなら `cd frontend && npx playwright test`（`dist` が e2e ビルドである必要がある）。

特定のシナリオだけ回す:

```bash
cd frontend && npx playwright test --grep "名前で部分一致検索する"
```

## 初回セットアップ

```bash
cd frontend && npm install
npx playwright install chromium
```

`npx playwright install --with-deps` は Debian 系専用なので、AlmaLinux では使えない。
AlmaLinux 10 (WSL2) では追加の `dnf` パッケージなしで headless Chromium が起動した。
他環境で起動しない場合は、次のスクリプトで不足ライブラリを確認する。

```bash
cd frontend && node -e "import('@playwright/test').then(async m => { const b = await m.chromium.launch(); await b.close(); console.log('ok'); })"
```

## 仕組み

```
Playwright (headless Chromium)
   │  page.goto / click / fill
   ▼
React (vite build --mode e2e)
   │  bindings/…/internal/service への import を shim に差し替え
   ▼
frontend/e2e/bindings-shim/   POST /api/{Service}/{Method}
   ▼
cmd/e2e-server                reflect でサービスに転送
   ▼
internal/service → internal/storage/sqlite → 一時ディレクトリの SQLite
```

- `vite-plugin-e2e-bindings.ts` が `resolveId` で生成 bindings を shim に解決し直す。
  有効になるのは `--mode e2e` のときだけで、`task dev` と `task build` には影響しない。
- shim は生成 bindings と同じ export 名とシグネチャを持つ。ずれると `npm run type-check` が落ちる
  (`e2e/bindings-shim/index.ts` の `_typeCheck`)。
- サーバと DB はテストごとに作り直す。`test.use({ seed: true })` を書いた describe だけサンプルレシピ入りで起動する。
- ルータは memory history なので `page.goto("/recipes")` では遷移しない。サイドバーをクリックする。

## シナリオとの対応

テスト名は `docs/scenarios/recipe.feature` の `シナリオ:` 行と一字一句同じにする。
`describe` 名は `機能:` 行に揃える。
未実装のシナリオは `test.skip` で名前だけ置く。

対応が取れているかは次で確認できる。

```bash
python3 - <<'PY'
import pathlib, re
feature = pathlib.Path("docs/scenarios/recipe.feature").read_text()
scenarios = [l.split("シナリオ:", 1)[1].strip() for l in feature.split("\n") if l.strip().startswith("シナリオ:")]
specs = "".join(p.read_text() for p in pathlib.Path("frontend/e2e").glob("*.spec.ts"))
names = set(re.findall(r'test(?:\.skip)?\(\s*"([^"]+)"', specs))
print("未対応シナリオ:", [s for s in scenarios if s not in names])
PY
```

## 新しいサービスを足したとき

1. `cmd/e2e-server/main.go` の `newRegistry` にサービスを追加する
2. `frontend/e2e/bindings-shim/` に同名のモジュールを足し、`index.ts` から re-export する
3. `npm run type-check` を通す (生成 bindings とのずれはここで落ちる)

## 検証していない範囲

Wails の IPC 層そのもの (メソッド ID ハッシュ、chunked 転送、イベント配信) は shim を通らない。
日本語 IME の入力も Playwright の `fill()` では再現しない。
どちらも Windows ビルドの目視確認に任せる (CLAUDE.md「動作確認の 2 階建て構成」)。
