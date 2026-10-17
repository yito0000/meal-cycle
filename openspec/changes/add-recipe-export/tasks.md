## 1. ドメイン層 (ファイルの中身とファイル名)

- [x] 1.1 `internal/domain/recipe/importfile.go` を更新し、コードブロックのフェンス行かを判定する処理を関数に切り出して `splitImportBody` から使う (design.md D3)。`go test ./internal/domain/recipe/...` が既存のまま通ることで確認
- [x] 1.2 `internal/domain/recipe/exportfile.go` を新設し、`ExportResult` struct を定義する (design.md D2)。`go build ./internal/...` で確認
- [x] 1.3 `internal/domain/recipe/exportfile.go` に `FormatExportFile(r Recipe) string` を実装する (design.md D3)。1.5 のテストで確認
  - frontmatter は yaml タグ付き struct を `yaml.v3` の `Marshal` で書く
  - 本文は空でない手順とメモだけを `## 手順` / `## メモ` の節で書く
  - コードブロックの外の行頭 `## ` を `### ` に下げる
- [x] 1.4 `internal/domain/recipe/exportfile.go` に `ExportFileName(name string, used map[string]bool) string` を実装する (design.md D4)。1.6 のテストで確認
- [x] 1.5 `internal/domain/recipe/exportfile_test.go` を新設し、`FormatExportFile` の table-driven test を書く。`go test ./internal/domain/recipe/...` で確認
  - 名前、人数、タグ、材料、手順、メモのあるレシピが `ParseImportFile` で同じ値に読み戻せる
  - タグと材料が空なら frontmatter に `tags` と `ingredients` が出ない
  - 手順だけ、メモだけ、両方空のときに、それぞれ該当する節だけが出る
  - 量 0 で単位「少々」の材料が、量 0、単位「少々」に読み戻せる
  - 量 0.5 と 0.33 の材料が同じ量で読み戻せる
  - `:` と `#` と先頭の `-` を含むレシピ名とタグが、そのまま読み戻せる
  - 手順の中の「## 下ごしらえ」が「### 下ごしらえ」になり、その後ろの手順も読み戻せる
  - コードブロックの中の `## ` の行は下げない
  - メモの中の「## 補足」も「### 補足」に下がる
- [x] 1.6 `internal/domain/recipe/exportfile_test.go` に `ExportFileName` の table-driven test を書く。`go test ./internal/domain/recipe/...` で確認
  - 「肉じゃが」は「肉じゃが.md」になる
  - 「鶏/豚の生姜焼き」は「鶏_豚の生姜焼き.md」になる
  - `< > : " \ | ? *` と制御文字がすべて `_` になる
  - 前後の空白と末尾の `.` が除かれる
  - 100 文字を超える名前が 100 文字に切られる
  - 置き換えの結果が空なら「recipe.md」になる
  - 「CON」と「com1」は先頭に `_` が付く
  - 同じ名前が 2 回来ると 2 回目は「カレー-2.md」、3 回目は「カレー-3.md」になる
  - 「Curry」のあとの「curry」は「curry-2.md」になる

## 2. リポジトリ層

- [x] 2.1 `internal/storage/sqlite/recipe_repo.go` を更新し、削除済みを除く全レシピを材料、タグ、手順、メモごと作成日の昇順で返す `ListDetailed` を足す (design.md D2)。2.2 のテストで確認
- [x] 2.2 `internal/storage/sqlite/recipe_repo_test.go` に `ListDetailed` のテストを足す。`go test ./internal/storage/sqlite/...` で確認
  - レシピが無いと空のスライスを返す
  - 材料、タグ、手順、メモが埋まった状態で返る
  - 削除済みのレシピは返らない
  - 作成日の昇順に並ぶ

## 3. サービス層

- [x] 3.1 `internal/service/folderpicker.go` を新設し、`FolderPicker` interface と、`application.Get().Dialog.OpenFile()` でフォルダだけを選ばせる `NewWailsFolderPicker` を実装する (design.md D1)。`go build .` で確認
- [x] 3.2 `internal/service/recipe.go` を更新し、`RecipeService` に `picker FolderPicker` を足して `NewRecipeService(db, picker)` にする。`go build ./internal/...` で確認
- [x] 3.3 `internal/service/recipe.go` に `ErrRecipeExportEmpty` と `ErrRecipeExportWrite` を足す (design.md D6)。`go build ./internal/...` で確認
- [x] 3.4 `internal/service/recipe.go` に `ExportToFolder(ctx, dialogTitle string) (recipe.ExportResult, error)` を実装する (design.md D2、D5)。3.6 のテストで確認
  - 書き出し用のフォルダ名は `meal-cycle-recipes-YYYYMMDD-HHMMSS` とし、既存なら `-2`、`-3` を付ける
  - 失敗したら作ったフォルダを消し、`ErrRecipeExportWrite` を包んで返す
- [x] 3.5 `internal/service/recipe_test.go` を更新し、`newRecipeServiceForTest` で picker を渡せるようにする。`go test ./internal/service/...` が既存のまま通ることで確認
- [x] 3.6 `internal/service/recipe_test.go` に `TestRecipeService_ExportToFolder` を BDD 風に足す。`go test ./internal/service/... -run TestRecipeService_ExportToFolder` で確認
  - should write every active recipe as one file when a folder is picked (Scenario「全レシピがフォルダに書き出される」)
  - should skip archived recipes when some are deleted (Scenario「削除済みのレシピは書き出されない」)
  - should write nothing and report cancelled when the picker returns empty (Scenario「フォルダ選択を閉じると何も書き出さない」)
  - should return ErrRecipeExportWrite and leave the picked folder unchanged when it is not writable (Scenario「書き込めないと失敗を知らせる」)
  - should return ErrRecipeExportWrite when the picker fails
  - should return ErrRecipeExportEmpty without opening the picker when no recipes exist
  - should create a suffixed folder when the timestamped folder already exists
  - should number files when two recipes share a name (Scenario「同じ名前のレシピは番号を付けて別のファイルになる」)
  - should replace characters unusable in file names (Scenario「ファイル名に使えない文字は置き換えられる」)
  - should produce files that PreviewImport reads back without issues and with the same contents

## 4. アプリと E2E サーバの配線

- [x] 4.1 `main.go` を更新し、`service.NewRecipeService(db, service.NewWailsFolderPicker())` で登録する。`go build .` で確認
- [x] 4.2 `cmd/e2e-server/main.go` を更新し、`-export-dir` フラグと、そのパスを返す `fixedFolderPicker` で `RecipeService` を作る。`go build ./cmd/...` で確認
- [x] 4.3 `cmd/e2e-server/dispatch_test.go` を `NewRecipeService` の引数に追従させる。`go test ./cmd/...` で確認
- [x] 4.4 `wails3 generate bindings -clean=true -ts -i` でバインディングを再生成する。`frontend/bindings/` に `ExportToFolder` と `ExportResult` が出ることで確認

## 5. フロントエンド

- [x] 5.1 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に `pages.recipes.export` の `button` / `dialog_title` / `done` / `empty` / `failed` を足す。`npm run type-check` で確認
- [x] 5.2 `frontend/src/components/recipe/recipe-export-button.tsx` を新設し、`useMutation` で `ExportToFolder` を呼んで結果に応じたトーストを出す (design.md D7)。`npm run type-check` で確認
- [x] 5.3 `frontend/src/routes/recipes.tsx` を更新し、ヘッダの `RecipeImportButton` の左に `RecipeExportButton` を置き、レシピが 0 件なら `disabled` を渡す。`task dev` の S02 で目視確認

## 6. E2E

- [x] 6.1 `frontend/e2e/fixtures.ts` を更新し、テストごとの一時ディレクトリの下に `exports` を作って `-export-dir` で渡し、`exportDir` fixture として公開する。既存の E2E が通ることで確認
- [x] 6.2 `frontend/e2e/recipe-export.spec.ts` を新設し、次の Scenario を書く。`task test:e2e` で確認
  - describe「S02 からのレシピのエクスポート」
    - 「全レシピがフォルダに書き出される」：`exportDir` の下にできたフォルダのファイル名と、トーストの件数とパスを確かめる
    - 「レシピが 0 件ならエクスポートできない」
  - describe「エクスポートするファイルの形式」
    - 「書き出したファイルはインポートで元と同じレシピになる」：書き出したファイルを `setFiles` でインポートし、S03 で内容を確かめる

## 7. ドキュメント

- [x] 7.1 `docs/recipe-import.md` に書き出しの節を足す (書き出し先のフォルダ、ファイル名の規則、見出しを下げること、100 件を超えたときの読み戻し方)。目視確認
- [x] 7.2 `docs/product.md` §4 S02 に「エクスポート」ボタンの挙動を足す。目視確認
- [x] 7.3 `docs/wireframes/S02_recipe_list.drawio.svg` のヘッダに「エクスポート」ボタンを足す。可視 SVG と `content=` の編集 XML の両方を直し、ブラウザと drawio の両方で開いて確認

## 8. 最終確認

- [x] 8.1 `go test ./internal/... ./cmd/...` がすべて pass することを確認
- [x] 8.2 `cd frontend && npm run type-check` と `cd frontend && npm run build:dev` が通ることを確認
- [x] 8.3 `task test:e2e` がすべて pass することを確認
- [x] 8.4 `task lint` が通ることを確認
- [ ] 8.5 `task build:win` で Windows ビルドを作り、実機でフォルダ選択、キャンセル、日本語名のファイルの書き出し、書き出したファイルのインポートを確かめる
