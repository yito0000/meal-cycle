## 1. DB の開き方 (リポジトリ層)

- [ ] 1.1 `internal/storage/sqlite/agent.go` を新設し、`ErrDatabaseMissing` / `ErrSchemaOutdated` / `ErrSchemaTooNew` を定義する。完了判定: `go build ./internal/...` が通る
- [ ] 1.2 `internal/storage/sqlite/agent.go` に `OpenExisting(path string) (*sql.DB, error)` を design.md D3 の手順で実装する。
  最新の版は埋め込んだ migration の最大の版から求め、スキーマの更新と控えの作成は行わない。完了判定: `go build ./internal/...` が通る
- [ ] 1.3 `internal/storage/sqlite/agent_test.go` を新設する。完了判定: `go test ./internal/storage/sqlite/ -run TestOpenExisting` が pass する
    - ファイルが無いと ErrDatabaseMissing を返し、ファイルを作らない
    - 古い版の DB に ErrSchemaOutdated を返し、版も backups フォルダも変わらない
    - 埋め込んだ最大より新しい版の DB に ErrSchemaTooNew を返す
    - 最新の版の DB を開いて返す

## 2. MCP サーバーの骨組み

- [ ] 2.1 `go get github.com/modelcontextprotocol/go-sdk@latest` で `go.mod` と `go.sum` を更新する。完了判定: `go.mod` の require に `github.com/modelcontextprotocol/go-sdk` がある
- [ ] 2.2 `cmd/mcp-server/main.go` を新設し、`-db` 引数 (省略時は `sqlite.DefaultPath()`) を読んで、stdio の transport で MCP サーバーを動かす。
  ログは標準エラーにだけ出す。完了判定: `go build ./cmd/mcp-server` が通る
- [ ] 2.3 `cmd/mcp-server/tools.go` を新設し、呼び出しごとに `OpenExisting` → `service.NewSettingsService` → 対象のサービスを作り、終わりに DB を閉じる共通の処理を書く (design.md D4)。
  `RecipeService` には何も選ばない `FolderPicker` を渡す。完了判定: `go build ./cmd/mcp-server` が通る
- [ ] 2.4 `cmd/mcp-server/errors.go` を新設し、domain の sentinel error と、`sqlite.ErrRecipeNotFound` / `ErrDatabaseMissing` などリポジトリ層の error をコード (`<capability>.<snake_case>`) に対応づける表と、
  ツールの失敗の結果を作る関数を書く (design.md D6)。表に無いエラーは `internal` にする。完了判定: `go build ./cmd/mcp-server` が通る

## 3. ツール

- [ ] 3.1 `cmd/mcp-server/tools.go` に参照のツール `list_recipes` / `get_recipe` / `list_meal_plan` / `list_stock` / `list_staples` / `get_shopping_list` を足す (design.md D5)。
  `list_recipes` は ID / 名前 / 人数 / タグだけを返す。完了判定: `go build ./cmd/mcp-server` が通る
- [ ] 3.2 `cmd/mcp-server/tools.go` に `list_suggestions` / `suggest_week_plans` を足す。完了判定: `go build ./cmd/mcp-server` が通る
- [ ] 3.3 `cmd/mcp-server/tools.go` に `assign_recipe` を足し、`meal` の入力スキーマを `breakfast` / `lunch` / `dinner` の enum にする。完了判定: `go build ./cmd/mcp-server` が通る
- [ ] 3.4 `cmd/mcp-server/tools.go` に `add_stock_items` を足し、`FridgeService.AddMany` を呼ぶ。完了判定: `go build ./cmd/mcp-server` が通る
- [ ] 3.5 `cmd/mcp-server/tools.go` に `add_staple` と `set_staple_state` を足し、`state` の入力スキーマを `enough` / `low` / `out` の enum にする。完了判定: `go build ./cmd/mcp-server` が通る
- [ ] 3.6 `cmd/mcp-server/tools.go` に `create_recipe` を足す。完了判定: `go build ./cmd/mcp-server` が通る
- [ ] 3.7 各ツールに英語の説明文を付け、戻り値を structured content と JSON テキストの両方で返す (design.md D5)。完了判定: `go build ./cmd/mcp-server` が通る

## 4. MCP サーバーのテスト

- [ ] 4.1 `cmd/mcp-server/tools_test.go` を新設し、`t.TempDir()` の DB を `sqlite.OpenAndMigrate` で作り、in-memory の transport でクライアントを繋ぐ `setup` を書く。
  完了判定: `go test ./cmd/mcp-server/` がコンパイルできる
- [ ] 4.2 `TestMCPServer_ListTools` を書く。完了判定: `go test ./cmd/mcp-server/ -run TestMCPServer_ListTools` が pass する
    - should return exactly the 13 tools in design.md D5 when the client lists tools
    - should not expose any delete or update tool when the client lists tools
- [ ] 4.3 `TestMCPServer_DatabaseCheck` を書く。完了判定: `go test ./cmd/mcp-server/ -run TestMCPServer_DatabaseCheck` が pass する
    - should fail with database.missing and not create the file when the database does not exist
    - should fail with database.schema_outdated and keep the version when the schema is old
    - should fail with database.schema_too_new when the schema is newer than the server
- [ ] 4.4 `TestMCPServer_Read` を書く。完了判定: `go test ./cmd/mcp-server/ -run TestMCPServer_Read` が pass する
    - should return the assigned entry when listing a week of meal plan
    - should fail with mealplan.invalid_period_weeks when weeks is 3
    - should fail with recipe.not_found when getting an unknown recipe
    - should return recipes without ingredients and steps when listing recipes
- [ ] 4.5 `TestMCPServer_AssignRecipe` を書く。完了判定: `go test ./cmd/mcp-server/ -run TestMCPServer_AssignRecipe` が pass する
    - should append the entry after existing ones when the meal already has an entry
    - should fail and keep the meal plan when the recipe does not exist
    - should fail with mealplan.invalid_date and keep the meal plan when the date is "11/2"
- [ ] 4.6 `TestMCPServer_AddStockItems` を書く。完了判定: `go test ./cmd/mcp-server/ -run TestMCPServer_AddStockItems` が pass する
    - should add all items when every item is valid
    - should fail with fridge.stock_name_empty and add nothing when one item has an empty name
- [ ] 4.7 `TestMCPServer_Staples` を書く。完了判定: `go test ./cmd/mcp-server/ -run TestMCPServer_Staples` が pass する
    - should list the staple in the shopping list when its state is set to out
    - should fail with fridge.staple_duplicate when adding a staple with an existing name
- [ ] 4.8 `TestMCPServer_CreateRecipe` を書く。完了判定: `go test ./cmd/mcp-server/ -run TestMCPServer_CreateRecipe` が pass する
    - should return the created recipe and include it in list_recipes when the input is valid
- [ ] 4.9 `TestMCPServer_Settings` を書く。完了判定: `go test ./cmd/mcp-server/ -run TestMCPServer_Settings` が pass する
    - should judge expiry status with the saved red days when settings are saved after the server started

## 5. GUI の取り直し

- [ ] 5.1 `frontend/src/main.tsx` を更新し、`focusManager.setEventListener` で `window` の `focus` と `visibilitychange` の両方からフォーカスを知らせる (design.md D7)。
  完了判定: `cd frontend && npm run type-check` が通る
- [ ] 5.2 `frontend/e2e/agent-access.spec.ts` を新設し、E2E サーバーの API で献立を割り当ててから `window` の `focus` を発火させるテストを書く。
  最外の describe は「GUI へのエージェントの書き込みの反映」にする。完了判定: `task test:e2e` で pass する
    - ウィンドウに戻ると割り当てた献立が表示される

## 6. ビルドとドキュメント

- [ ] 6.1 `Taskfile.yml` に `build:mcp` を足す (design.md D8)。完了判定: `task build:mcp` で `bin/meal-cycle-mcp` ができる
- [ ] 6.2 `Taskfile.yml` の `build:win` に `meal-cycle-mcp.exe` のビルドを足す。完了判定: `task build:win` で `bin/meal-cycle-mcp.exe` ができる
- [ ] 6.3 `Taskfile.yml` の `build:win:copy` で `meal-cycle-mcp.exe` も `WIN_DEST` へコピーする。完了判定: `WIN_DEST` を指定した `task build:win` でコピー先に 2 つの exe がある
- [ ] 6.4 `docs/mcp.md` を新設し、ビルド、Claude Desktop への登録例、公開するツールの一覧、エージェントに渡るデータの注意を書く。完了判定: 目視
- [ ] 6.5 `docs/design.md` の §1 想定シナリオ、§2 通信の行、§6 アーキテクチャに MCP サーバーを足す。完了判定: 目視
- [ ] 6.6 `docs/product.md` の §1 ナラティブに、エージェントに献立を手伝わせる使い方を 1 段落足す。完了判定: 目視
- [ ] 6.7 `CLAUDE.md` の常用コマンドに `task build:mcp`、アーキテクチャに MCP の入口を足す。完了判定: 目視
- [ ] 6.8 `README.md` に MCP サーバーの節を足し、`docs/mcp.md` へリンクする。完了判定: 目視

## 7. 確認

- [ ] 7.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全 pass
- [ ] 7.2 `cd frontend && npm run test` を実行する。完了判定: 全 pass
- [ ] 7.3 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0
- [ ] 7.4 `cd frontend && npm run build:dev` を実行する。完了判定: 成功
- [ ] 7.5 `task test:e2e` を実行する。完了判定: 全 pass
- [ ] 7.6 `task lint` を実行する。完了判定: エラー 0
- [ ] 7.7 Windows 実機で Claude Desktop に `meal-cycle-mcp.exe` を登録し、ツールの一覧が出ることと `assign_recipe` で足した品が GUI に戻ったとき表示されることを確かめる。完了判定: 目視
