## Why

フロントエンドに自動テストが 1 本も無い。
S02 レシピ一覧の検索とソート、S04 の入力バリデーションと未保存確認ダイアログは、すべて手動目視でしか確認できていない。
これらのロジックは Go 側ではなく React 側 (`recipe-sort.ts` / `use-recipe-form.ts`) にあるため、
既存のサービス層テストではカバーできない。

一方で Wails v3 の webview は外部のテストドライバから掴めない。
Linux 側は WebKit2GTK で CDP を持たない。
`wails3 dev` でも webview が読むのはカスタムスキーム経由の asset server なので、Playwright を後ろから接続する口が無い。

そこで Wails の IPC 層だけを HTTP に差し替え、React と Go サービスと SQLite は本物のまま通す E2E 基盤を用意する。
WSL2 上でヘッドレス実行できるため、日本語入力の問題 (WSLg + WebKit2GTK の IME) を回避したまま反復できる。

## What Changes

- `cmd/e2e-server` を新設する。`frontend/dist` を配信し、`POST /api/{Service}/{Method}` を受けて
  `internal/service` のメソッドにディスパッチする単一バイナリ。DB パスと待受ポートはフラグで受ける。
- ディスパッチはリフレクションで行う。引数は JSON 配列で受け、メソッドの各引数の型に unmarshal する。
  第 1 引数の `context.Context` と戻り値末尾の `error` はいずれも省略可として扱う
  (`HealthService.Ping()` は ctx を取らず error も返さない)。
- `error` を返すメソッドがエラーを返した場合は HTTP 500 + `{"kind": ..., "message": ...}` で返し、
  shim 側で `Error` を throw する。
  Wails 既定トランスポートのエラー形状 (`@wailsio/runtime` の `runtime.js`) に合わせ、フロントの `catch` を書き換えずに済ませる。
- `frontend/e2e/bindings-shim/` に生成 bindings と同じ export を持つ fetch 版モジュールを置く。
- `frontend/vite-plugin-e2e-bindings.ts` を新設し、`resolveId` フックで `bindings/github.com/.../internal/service` への
  import を shim に解決し直す。生成 bindings は相対パスで import されるため alias では届かない。
- プラグインは `mode === "e2e"` のときだけ有効化する。既定のビルドとホットリロードには一切影響しない。
- `@playwright/test` を `frontend/` の devDependency に追加し、`frontend/playwright.config.ts` と
  `frontend/e2e/*.spec.ts` を置く。ブラウザは headless Chromium 1 種類に絞る。
- Playwright の worker 単位 fixture で `cmd/e2e-server` を起動する。DB は `t.TempDir` 相当の一時ディレクトリに作り、
  テストファイルごとに使い捨てる。マイグレーションはサーバ起動時に走らせる。
- `cmd/seed/main.go` の `sampleRecipes()` を `internal/sampledata` に切り出し、seed と E2E fixture で共有する。
  ディレクトリ名に `testdata` を使わないのは、Go ツールがその名前をワイルドカードから除外し lint が効かなくなるため。
- テスト名を `docs/scenarios/*.feature` の `シナリオ:` 行と一致させ、シナリオと自動テストの対応を追えるようにする。
- 実装済みなのに `docs/scenarios/recipe.feature` にシナリオが無い経路 (名前昇順の並べ替え、材料の自由テキスト入力、
  バリデーションエラー表示、タグ追加) にシナリオを追加し、テストと 1 対 1 に対応させる。
- `docs/scenarios/README.md` の「自動テストとしての実行について」を書き換え、godog 見送りの判断と本基盤への置き換えを記録する。
- `CLAUDE.md` の「テスト方針」に E2E 層を追記し、`task test:e2e` を常用コマンドに加える。

### Non-goals

- **godog の導入**：見送る。サービス層は既に `internal/service/*_test.go` が `t.Run("should ... when ...")` の BDD 風で
  書かれており、godog を足しても得られるのは Gherkin の見た目だけでカバー範囲は変わらない。さらに「検索欄に入力する」型の
  シナリオを Go のステップに写すと、実際にはフロントにある検索とソートを Go 側で再実装した嘘の検証になる。
  `.feature` は仕様ドキュメントとして残し、テスト名の対応でトレーサビリティを取る。
- **Vitest / Testing Library の導入**：純関数とコンポーネント単体の層は別 proposal `add-frontend-unit-tests` で扱う。
  本 proposal は画面を通した経路だけを対象にする。
- **Windows ビルドに対するスモークテスト**：Windows 版は WebView2 なので `--remote-debugging-port` 経由で
  `connectOverCDP` できる見込みがあるが、未検証かつ CI に載らない。別 proposal `add-windows-smoke-e2e` で扱う。
- **Wails IPC 層そのものの検証**：`Call.ByID` のメソッド ID ハッシュ、chunked 転送、イベント配信は shim を通らないため
  対象外。ここは引き続き Windows 実機での目視確認に任せる。
- **CI (GitHub Actions) の追加**：本 proposal はローカルで `task test:e2e` が完走するところまで。リポジトリに `.github/` は
  まだ無く、CI の設計は別途行う。
- **fridge / shopping capability のシナリオ**：画面が未実装なので書けない。各 capability の proposal 側で追加する。
- **`MealPlanService` の永続化**：現状 in-memory のまま。献立の E2E は固定データ前提で書き、SQLite 化は別 proposal で扱う。
- **視覚回帰テスト**：スクリーンショット比較は入れない。DOM とテキストに対するアサーションのみ。

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

(なし)

本 proposal は開発ツーリングの追加であり、プロダクトの観察可能な挙動は変えない。
`.openspec.yaml` に `skip_specs: true` を設定し、spec delta は作らない。
受け入れ条件は tasks.md 側に完了判定として書く。

docs/design.md §8 は「スキャフォールド系は proposal にしない」としている。
本件は実現方法に複数の選択肢があり、選定理由を残す価値が大きいため、例外として proposal 化する。

## Impact

### 新設ファイル

- `cmd/e2e-server/main.go`：フラグ解析、SQLite オープンとマイグレーション、`http.Server` の起動
- `cmd/e2e-server/dispatch.go`：サービス名からメソッドを引くレジストリと、リフレクション呼び出し
- `cmd/e2e-server/dispatch_test.go`：ディスパッチャの単体テスト
- `internal/sampledata/recipes.go`：`cmd/seed` から切り出したサンプルレシピ
- `frontend/vite-plugin-e2e-bindings.ts`：bindings を shim に解決し直す Vite プラグイン
- `frontend/e2e/bindings-shim/service.ts`：`RecipeService` / `MealPlanService` / `HealthService` の fetch 実装
- `frontend/e2e/bindings-shim/index.ts`：生成 bindings の `index.ts` と同じ re-export
- `frontend/e2e/fixtures.ts`：`cmd/e2e-server` を一時 DB で起動する Playwright fixture
- `frontend/e2e/recipe-listing.spec.ts`：S02 のシナリオ
- `frontend/e2e/recipe-edit.spec.ts`：S04 のシナリオ
- `frontend/playwright.config.ts`：headless Chromium、`webServer` 設定、レポータ

### 更新ファイル

- `cmd/seed/main.go`：`sampleRecipes()` の定義を削除し `internal/sampledata` を参照する
- `frontend/vite.config.ts`：`mode === "e2e"` のときだけ E2E プラグインを `plugins` に追加する
- `frontend/package.json`：`@playwright/test` を devDependencies に追加、`test:e2e` スクリプトを追加
- `frontend/tsconfig.json`：`include` に `e2e` と `vite-plugin-e2e-bindings.ts` を追加する
  (追加しないと shim の型検査が空振りする)
- `frontend/biome.json`：`includes` に `e2e/**/*.ts` と `vite.config.ts` / `vite-plugin-e2e-bindings.ts` /
  `playwright.config.ts` を追加する (追加前はいずれも lint 対象外だった)
- `Taskfile.yml`：`test:e2e` タスクを追加、`lint:go` と `format:go` は `./cmd/...` を既に含む
- `.gitignore`：`frontend/playwright-report/` と `frontend/test-results/` を追加
- `docs/scenarios/README.md`：「自動テストとしての実行について」を書き換える
- `docs/scenarios/recipe.feature`：実装済みだがシナリオが無い並べ替えと保存の経路にシナリオを追加する
- `CLAUDE.md`：「テスト方針」に E2E 層、「常用コマンド」に `task test:e2e` を追記

### 生成物 (再生成のみ)

- `frontend/bindings/`：`wails3 generate bindings -clean=true -ts -i`。
  素で叩くと `.js` が出るため、`build/Taskfile.yml` の `generate:bindings` と同じフラグを付ける

### migration

なし。既存の `00001` / `00002` をそのまま使う。

### 後続 proposal への引き継ぎ

- 新しい capability の proposal を切るときは、tasks に「対応する `frontend/e2e/*.spec.ts` を追加する」を含める。
  shim も新サービス分の関数を足す必要があるため、サービス追加とセットで扱う。
- shim と生成 bindings のずれは `npm run type-check` で検出する。
  両者が同じ型を参照する構造にしておく (詳細は design.md)。
  ずれを機械的に検出できない形にした場合、後続 proposal で必ず壊れる。
- `MealPlanService` を SQLite 化する proposal では、`internal/sampledata` に献立のサンプルを足して
  `frontend/e2e/meal-plan.spec.ts` を追加する。
- CI を入れる proposal では、AlmaLinux 上の Playwright 依存パッケージ導入手順が課題になる (design.md の Risk 参照)。
