## 1. テストデータの共有化

- [x] 1.1 `internal/sampledata/recipes.go` を新設し、`cmd/seed/main.go` の `sampleRecipes()` を
      `sampledata.Recipes() []recipe.Recipe` として移設する。
      ディレクトリ名を `testdata` にしないのは、Go ツールがその名前をワイルドカードから外し lint 対象外になるため。
      完了判定: `go build ./internal/... ./cmd/... ./` が通る
- [x] 1.2 `cmd/seed/main.go` を更新し、自前の `sampleRecipes()` 定義を削除して `sampledata.Recipes()` を参照する。
      完了判定: `task seed -- -db /tmp/seed-check.db` が成功し、2 回目の実行が全件 skip でログ出力される

## 2. E2E サーバ (Go)

- [x] 2.1 `cmd/e2e-server/dispatch.go` を新設し、サービス名からレシーバを引く `map[string]any` のレジストリを定義する。
      登録するのは `HealthService` / `MealPlanService` / `RecipeService` の 3 つ (`main.go` と同じ構成)。
      完了判定: `go build ./cmd/...` が通る
- [x] 2.2 `cmd/e2e-server/dispatch.go` に `POST /api/{Service}/{Method}` のハンドラを実装する。
      ボディを `[]json.RawMessage` として受け、メソッドの各引数の `reflect.Type` に個別に unmarshal して
      `reflect.Value.Call` する。第 1 引数が `context.Context` の場合はそこに `r.Context()` を渡す。
      完了判定: 2.4 のテストが通る
- [x] 2.3 `cmd/e2e-server/dispatch.go` にエラー応答を実装する。戻り値末尾が `error` で非 nil なら HTTP 500 と
      `{"kind": "<エラー型名>", "message": "<Error()>"}` を返し、未知のサービス名とメソッド名は HTTP 404 を返す。
      `error` を返さないメソッドは戻り値をそのまま JSON で返す。
      形状は `@wailsio/runtime` の `dist/runtime.js` が期待するものに合わせる。
      完了判定: 2.4 のテストが通る
- [x] 2.4 `cmd/e2e-server/dispatch_test.go` を新設し、ディスパッチャの単体テストを書く
  - 引数 0 個のメソッド (`RecipeService.List`) を呼ぶと 200 と JSON 配列が返る
  - 引数 1 個のメソッド (`RecipeService.GetByID`) に文字列引数を渡すと 200 と単一オブジェクトが返る
  - 引数 2 個のメソッド (`RecipeService.Update`) に文字列と構造体を渡すと 200 が返る
  - 構造体引数 (`RecipeService.Create`) のネストしたフィールド (Ingredients) が欠落せず Go 側に届く
  - サービスがエラーを返すと 500 と `kind` / `message` を含む JSON が返る
  - 引数の個数がメソッドと一致しないと 400 が返る
  - ボディが JSON として不正なら 400 が返る
  - 未知のサービス名は 404 が返る
  - 未知のメソッド名は 404 が返る
  - ctx を取らず error も返さないメソッド (`HealthService.Ping`) を呼ぶと 200 と JSON が返る
  - レジストリに登録されたサービスが `main.go` の 3 つと一致する
- [x] 2.5 `cmd/e2e-server/main.go` を新設し、`-db` (SQLite パス) / `-port` (既定 0) / `-dist` (配信元) /
      `-seed` (bool) のフラグを定義する。`-dist` は任意の cwd から起動できるようにするため実装時に追加した。
      完了判定: `go run ./cmd/e2e-server -h` がこの 4 フラグを表示する
- [x] 2.6 `cmd/e2e-server/main.go` に起動処理を実装する。`sqlite.Open` と `sqlite.Migrate` を実行し、
      `-seed` が真なら `sampledata.Recipes()` を `RecipeService.Create` 経由で投入する。
      完了判定: `go run ./cmd/e2e-server -db /tmp/e2e-check.db -seed` が起動し、DB ファイルが作られる
- [x] 2.7 `cmd/e2e-server/main.go` に `frontend/dist` の静的配信を実装する。`/api/` 以外は
      `http.FileServer` に流し、見つからないパスは `index.html` を返す (TanStack Router の memory history 対策)。
      完了判定: `frontend` を `npm run build:dev` した状態でサーバに `/` を GET すると `index.html` が返る
- [x] 2.8 `cmd/e2e-server/main.go` を `-port 0` で listen させ、実際に割り当てられたポートを
      `E2E_SERVER_LISTENING http://127.0.0.1:<port>` の 1 行で stdout に出力する。
      完了判定: 起動ログにこの行が出て、その URL に `curl` で到達できる

## 3. bindings shim と Vite プラグイン (Frontend)

- [x] 3.1 `wails3 generate bindings -clean=true -ts -i` を実行し、`frontend/bindings/` を最新化する。
      フラグは `build/Taskfile.yml` の `generate:bindings` に揃える (素で叩くと `.ts` ではなく `.js` が出る)。
      完了判定: `frontend/bindings/github.com/yito0000/meal-cycle/internal/service/index.ts` が存在する
- [x] 3.2 `frontend/e2e/bindings-shim/service.ts` を新設し、`RecipeService` の
      `List` / `GetByID` / `Create` / `Update` を `POST /api/RecipeService/<メソッド名>` への `fetch` として実装する。
      戻り値は `@wailsio/runtime` の `CancellablePromise` を `new` して返す。
      完了判定: `npm run type-check` が通る
- [x] 3.3 `frontend/e2e/bindings-shim/service.ts` に `MealPlanService.ListWeek` と `HealthService` の公開関数を
      同じ方式で追加する。
      完了判定: `npm run type-check` が通る
- [x] 3.4 `frontend/e2e/bindings-shim/service.ts` に HTTP 500 応答のハンドリングを実装する。
      レスポンスの `kind` と `message` から `Error` を組み立てて reject する。
      完了判定: 5.7 のバリデーションエラーのテストが通る
- [x] 3.5 `frontend/e2e/bindings-shim/index.ts` の末尾に、生成 bindings との構造互換を検査する
      `const _typeCheck: typeof Generated = { ... }` を追加する (design.md D5 参照)。
      あわせて `frontend/tsconfig.json` の `include` に `e2e` を追加する (無いと型検査が空振りする)。
      完了判定: shim の関数を 1 つ意図的に削ると `npm run type-check` が落ち、戻すと通る
- [x] 3.6 `frontend/e2e/bindings-shim/index.ts` を新設し、生成 bindings の `index.ts` と同じ形で
      `service.ts` を re-export する。
      完了判定: `npm run type-check` が通る
- [x] 3.7 `frontend/vite-plugin-e2e-bindings.ts` を新設し、`resolveId` フック (`order: "pre"`) で
      解決先が `/bindings/github.com/yito0000/meal-cycle/internal/service` 配下になる import を
      `frontend/e2e/bindings-shim/` に振り替える。`internal/domain/*/models` は振り替えない。
      完了判定: 4.4 の疎通テストが通る
- [x] 3.8 `frontend/vite.config.ts` を更新し、`mode === "e2e"` のときだけ 3.7 のプラグインを `plugins` に追加する。
      完了判定: `npm run build:dev` の出力に shim が含まれず、`vite build --mode e2e` の出力には含まれる

## 4. Playwright のセットアップ

- [x] 4.1 `frontend/package.json` に `@playwright/test` と `@types/node` を devDependency として追加する。
      `@types/node` は fixture が `node:child_process` などを使うため型解決に必要。
      完了判定: `npm install` が成功し `npx playwright --version` が表示される
- [x] 4.2 AlmaLinux 10 上で `npx playwright install chromium` を実行し、不足する共有ライブラリがあれば `dnf` で補う。
      結果を `frontend/e2e/README.md` に記録する (実測では追加パッケージ不要だった)。
      完了判定: headless Chromium がページを開いてテキストを読めることを確認する
- [x] 4.3 `frontend/playwright.config.ts` を新設する。`testDir: "./e2e"`、プロジェクトは headless Chromium 1 つ、
      `fullyParallel: false`、`webServer` は使わない (design.md D7)。
      完了判定: `npx playwright test --list` がテスト 0 件でエラーなく終了する
- [x] 4.4 `frontend/e2e/fixtures.ts` を新設し、worker 単位で `cmd/e2e-server` を子プロセス起動する fixture を書く。
      一時ディレクトリに DB を作り、stdout の `E2E_SERVER_LISTENING` 行を待ってから `baseURL` を確定し、
      worker 終了時にプロセスと一時ディレクトリを片付ける。
      完了判定: サーバに到達して `/` が開けることだけを確認する疎通テストが 1 本通る
- [x] 4.5 `frontend/e2e/fixtures.ts` に `-seed` の有無を spec 側から切り替えられるオプションを追加する。
      完了判定: seed 有りの spec で 5.2 が通り、seed 無しの spec で 5.1 が通る
- [x] 4.6 `frontend/package.json` に `test:e2e` スクリプトを追加する
      (`vite build --mode e2e` の後に `playwright test` を走らせる)。
      完了判定: `npm run test:e2e` が疎通テストを実行して成功する
- [x] 4.7 `.gitignore` に `frontend/playwright-report/` と `frontend/test-results/` を追加する。
      完了判定: `npm run test:e2e` の後に `git status` がこの 2 つを表示しない

## 5. E2E シナリオ

- [x] 5.1 `docs/scenarios/recipe.feature` に、実装済みだがシナリオが無い 5 件を追加する
  - 名前 昇順に並べ替える
  - 検索に一致するレシピが無い場合は空状態が表示される
  - 材料を自由テキストで入力してレシピを登録する
  - レシピ名を空のまま保存するとエラーが表示される
  - タグを追加して保存すると一覧のタグ列に反映される
- [x] 5.2 `frontend/e2e/recipe-listing.spec.ts` を新設し、`describe` 名を `機能: レシピの一覧と詳細閲覧` にする。
      アサーションの文言は `src/locales/ja.json` を import して参照し、日本語をテストにハードコードしない。
      完了判定: `npx playwright test recipe-listing --list` が想定のテスト名を並べる
- [x] 5.3 `frontend/e2e/recipe-listing.spec.ts` に一覧のテストを実装する (テスト名は `.feature` の `シナリオ:` と一字一句同じ)
  - 登録済みレシピが 1 件もない状態で一覧を開く
  - 登録済みレシピを更新日 desc（デフォルト）で一覧表示する
  - 名前で部分一致検索する
  - 名前 昇順に並べ替える
  - 検索に一致するレシピが無い場合は空状態が表示される
- [x] 5.4 `frontend/e2e/recipe-listing.spec.ts` に、S03 未実装のシナリオを `test.skip` で名前だけ置く
  - レシピ一覧から詳細ドロワーを開く
  - 詳細ドロワーを背景クリックで閉じる
- [x] 5.5 `frontend/e2e/recipe-edit.spec.ts` を新設し、`describe` 名を `機能: レシピの登録と編集` にする。
      完了判定: `npx playwright test recipe-edit --list` が想定のテスト名を並べる
- [x] 5.6 `frontend/e2e/recipe-edit.spec.ts` に登録と編集のテストを実装する
  - 材料を自由テキストで入力してレシピを登録する
  - 既存レシピを編集して保存すると一覧に反映される
  - タグを追加して保存すると一覧のタグ列に反映される
  - 未保存でキャンセルすると確認される
- [x] 5.7 `frontend/e2e/recipe-edit.spec.ts` にバリデーションのテストを実装する
  - レシピ名を空のまま保存するとエラーが表示される
- [x] 5.8 `frontend/e2e/recipe-edit.spec.ts` に、食材マスタと S03 が未実装のシナリオを `test.skip` で名前だけ置く
  - 既存 Ingredient を材料にしてレシピを登録する
  - 新規 Ingredient を作りながらレシピを登録する
  - 詳細ドロワーから編集画面へ遷移し、保存後に一覧に戻る
- [x] 5.9 `docs/scenarios/recipe.feature` の全 `シナリオ:` 行と `frontend/e2e/*.spec.ts` の全テスト名を突き合わせ、
      対応が取れていない行が無いことを確認する。
      完了判定: `grep "シナリオ:" docs/scenarios/recipe.feature` の各行が spec のテスト名として存在する

## 6. Taskfile とドキュメント

- [x] 6.1 `Taskfile.yml` に `test:e2e` タスクを追加する
      (`cd frontend && npm run type-check` の後に `npm run test:e2e` を実行する)。
      完了判定: `task test:e2e` が完走する
- [x] 6.2 `frontend/e2e/README.md` を新設し、初回セットアップ手順 (4.2 の `dnf` パッケージ一覧を含む) と
      shim の仕組みを書く。
      完了判定: 手順どおりに実行して `task test:e2e` が通ることを別シェルで確認する
- [x] 6.3 `docs/scenarios/README.md` の「自動テストとしての実行について」を書き換え、godog を見送った理由と
      Playwright ベースの現行構成、テスト名で対応を取る運用を記述する。
      完了判定: 節の中に godog を導入しない判断とその理由が書かれている
- [x] 6.4 `CLAUDE.md` の「テスト方針」に E2E 層の説明を追記し、「常用コマンド」に `task test:e2e` を加える。
      完了判定: 該当 2 箇所に追記されている
- [x] 6.5 `CLAUDE.md` の「動作確認の 2 階建て構成」に、E2E が下段 (WSL 側) の自動化であり、
      Windows 実機確認を置き換えるものではないことを 1 行で追記する。
      完了判定: 該当節に追記されている

## 7. 最終確認

- [x] 7.1 `go test ./internal/... ./cmd/...` を実行し、全テストが通ることを確認する
- [x] 7.2 `cd frontend && npm run type-check` を実行し、エラーが無いことを確認する
- [x] 7.3 `cd frontend && npm run build:dev` を実行し、E2E プラグインが既定ビルドに影響していないことを確認する
- [x] 7.4 `task test:e2e` を実行し、`test.skip` を除く全 E2E テストが通ることを確認する
- [x] 7.5 `task lint` を実行し、Go と Frontend の両方が通ることを確認する
      (`frontend/biome.json` の `includes` に `e2e/` と各設定ファイルを追加。追加前は lint 対象外だった)
- [x] 7.6 `task dev` を起動し、ホットリロード開発モードが従来どおり動くことを確認する
      (bindings が `.ts` で再生成され、Vite dev が 9245 で応答し、ウィンドウが 2400x1500 で開くまでをログで確認。
      WSL の画面そのものの目視は未実施)
