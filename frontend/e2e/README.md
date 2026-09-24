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

E2E が確かめる振る舞いの正は `openspec/specs/<capability>/spec.md` にある。
対応はテスト名で取り、次の 3 つを一字一句揃える。

- ファイル名の先頭語: capability 名（`fridge.spec.ts`、`mealplan-batch.spec.ts` など `<capability>[-<topic>].spec.ts`）
- 最外の `describe` 名: `### Requirement:` の名前
- `test` 名: その Requirement 配下の `#### Scenario:` の名前

内側に `describe` を挟むのは構わない（`test.use({ seed: true })` を一部のテストにだけ効かせたいときなど）。
まだ E2E を書いていないシナリオは、`test.skip` で名前だけ置いてもよい。

spec.md の全シナリオに E2E を求めはしない。
サービス層のテストで確かめるシナリオ（「再起動後も在庫が残る」など）も spec.md に並ぶためである。
どのシナリオを E2E で確かめるかは、proposal の tasks.md で決める。

対応は `npm run check:scenarios` が検査し、`task test:e2e` の最初にも走る。
検査するのは「E2E のテスト名が spec.md に実在するか」の向きだけで、実装中の change の delta spec（`openspec/changes/<name>/specs/`）も照合先に含める。
最後に capability ごとの E2E 件数と Scenario 件数を表示する。

```bash
cd frontend && npm run check:scenarios
```

### Gherkin の実行系を入れない理由

当初は `docs/scenarios/*.feature` に Gherkin でシナリオを置き、[cucumber/godog](https://github.com/cucumber/godog) で直接実行する案を検討したが見送った。
Go のステップからは React の画面を操作できず、画面側の検索やソートを Go で書き直した検証になってしまうためである。
判断の詳細は [../../openspec/changes/archive/2026-09-07-add-e2e-harness/proposal.md](../../openspec/changes/archive/2026-09-07-add-e2e-harness/proposal.md) の Non-goals にある。

Playwright から `.feature` を実行する playwright-bdd も見送った。
シナリオのステップが自由記述のためステップ定義を使い回せず、文面をステップ定義に合わせて型にはめる必要が出るためである。
`.feature` 自体も、spec.md との二重管理で古くなったため廃止し、シナリオは spec.md に一本化した（[../../docs/product.md](../../docs/product.md) §6）。

## 新しいサービスを足したとき

1. `cmd/e2e-server/main.go` の `newRegistry` にサービスを追加する
2. `frontend/e2e/bindings-shim/` に同名のモジュールを足し、`index.ts` から re-export する
3. `npm run type-check` を通す (生成 bindings とのずれはここで落ちる)

## 検証していない範囲

Wails の IPC 層そのもの (メソッド ID ハッシュ、chunked 転送、イベント配信) は shim を通らない。
日本語 IME の入力も Playwright の `fill()` では再現しない。
どちらも Windows ビルドの目視確認に任せる (CLAUDE.md「動作確認の 2 階建て構成」)。
