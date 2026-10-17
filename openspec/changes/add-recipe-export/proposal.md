## Why

レシピはインポートで入れられるようになったが、アプリから外へ出す手段が無い。
データは手元の SQLite ファイルにしか無く、レシピを別の PC へ移したり、手で直してから入れ直したり、AI に読ませて献立の相談に使ったりするには、
ファイルとして取り出せる必要がある。
インポート用の形式 ([docs/recipe-import.md](../../../docs/recipe-import.md)) はすでに人が読み書きできる Markdown として決まっているので、
同じ形式で書き出せば、書き出したファイルをそのままインポートで読み戻せる。

## What Changes

- S02 レシピ一覧のヘッダ、「インポート」ボタンの左に「エクスポート」ボタンを置く
- ボタンを押すと OS のフォルダ選択ダイアログが開き、選んだフォルダの中に書き出し用のフォルダを 1 つ作って、登録済みのレシピをすべて書き出す
  - 削除済みのレシピは書き出さない
  - 1 レシピを 1 ファイルにし、ファイル名はレシピ名から作る
  - 形式はインポートと同じ (frontmatter に名前、人数、タグ、材料、本文に「## 手順」「## メモ」)
- 書き出したファイルをインポートで読み戻すと、元と同じ内容のレシピになる
  - 例外は 1 つだけで、手順とメモの中のレベル 2 の見出しはレベル 3 に下げて書き出す。インポートがレベル 2 の見出しを節の区切りとして読むため
- 書き出しが終わると、件数と書き出し先のフォルダをトーストで知らせる。ダイアログを閉じたときは何もしない
- 書き込みに失敗したときは失敗をトーストで知らせ、作りかけのフォルダを残さない
- レシピが 0 件のときはボタンを押せない
- [docs/recipe-import.md](../../../docs/recipe-import.md) に書き出しの節を足し、文書名に合わせて読み込みと書き出しの両方の形式の正とする

## Non-goals

- 献立、在庫、買い物リストの書き出し、DB 全体のバックアップと復元。S09 (設定) で扱う
- 書き出すレシピを選ぶこと (検索で絞った分だけ、チェックした分だけ、など)。常に全件を書き出す
- S03 から 1 件だけ書き出すこと
- 書き出し先の既存ファイルの上書き。毎回新しいフォルダを作るので、上書きは起こらない
- 書き出したフォルダを OS のファイルマネージャで開くこと
- インポートの上限 (100 件) を書き出しに合わせて広げること。Risks を参照

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `recipe`: S02 からのレシピのエクスポートと、書き出すファイルの形式 (インポートで読み戻せること) の Requirement を足す

## Impact

**新設ファイル:**

- `internal/domain/recipe/exportfile.go`：`FormatExportFile` (1 件を Markdown にする)、`ExportFileName` (レシピ名からファイル名を作る)、`ExportResult`
- `internal/domain/recipe/exportfile_test.go`
- `internal/service/folderpicker.go`：`FolderPicker` interface と、Wails のダイアログを使う実装
- `frontend/src/components/recipe/recipe-export-button.tsx`

**更新ファイル:**

- `internal/storage/sqlite/recipe_repo.go`：削除済みを除く全レシピを材料ごと返す `ListDetailed` を足す
- `internal/storage/sqlite/recipe_repo_test.go`
- `internal/service/recipe.go`：`RecipeService` に `FolderPicker` を持たせ、`ExportToFolder` を足す。`NewRecipeService` に picker の引数を足す
- `internal/service/recipe_test.go`
- `main.go`：`service.NewRecipeService(db, service.NewWailsFolderPicker())` に変える
- `cmd/e2e-server/main.go`：`-export-dir` フラグを足し、そのパスを返す picker で `RecipeService` を作る
- `cmd/e2e-server/dispatch_test.go`：`NewRecipeService` の引数の変更に追従する
- `frontend/src/routes/recipes.tsx`：ヘッダに `RecipeExportButton` を置く
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：`pages.recipes.export.*` を足す
- `frontend/e2e/recipe-export.spec.ts` (新設)
- `frontend/e2e/fixtures.ts`：テストごとの一時ディレクトリの下の `exports` を `-export-dir` で E2E サーバに渡し、`exportDir` fixture として公開する
- `docs/recipe-import.md`：書き出しの節を足す
- `docs/product.md` §4 S02：「エクスポート」ボタンの挙動を足す
- `docs/wireframes/S02_recipe_list.drawio.svg`：ヘッダに「エクスポート」ボタンを足す (可視 SVG と `content=` の編集 XML の両方)

**生成物 (再生成のみ):** `frontend/bindings/` (`wails3 generate bindings`)

**migration:** なし

**後続 proposal への引き継ぎ:**

- S09 (設定)：DB 全体のバックアップと復元を作るとき、レシピの書き出しはこの proposal の形式のまま残す。全データの書き出しは別の形式にする
- インポートの上限 100 件：レシピが 100 件を超えると、書き出したフォルダを 1 回では読み戻せない。困る件数になったら、上限か読み込みの分割を見直す
- 検索とタグの絞り込み (S10) が入ったら、絞り込んだ分だけを書き出す選択肢を検討する
