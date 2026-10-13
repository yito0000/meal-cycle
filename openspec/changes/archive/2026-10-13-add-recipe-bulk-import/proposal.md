## Why

レシピは S04 で 1 件ずつ手で入力するしかなく、材料の行を 1 つずつ足していく作業が登録の手間の大半を占める。
自分で作れるレシピには限りがあるので、一般的な家庭料理の代表的なものを料理サイトから集めて持っておきたい。
料理サイトのページを AI に読ませれば、レシピを決まった形式のテキストに変換すること自体はすぐにできる。
足りないのは、そのテキストをアプリにまとめて取り込む入口である。

アプリは完全ローカルで外部と通信しないので、ページの取得や AI での変換はアプリの外で行う。
アプリは変換済みのファイルを読んで登録するところだけを受け持つ。

## What Changes

- S02 に「インポート」ボタンを置き、Markdown ファイルを選ぶか、フォルダを選んでその中の Markdown ファイルをまとめて読み込めるようにする
- **インポート形式** を定める。1 レシピ = 1 Markdown ファイルで、先頭の YAML frontmatter に名前、人数、タグ、材料を書き、
  本文の「## 手順」と「## メモ」の節をそれぞれ手順とメモにする
  - 元レシピの URL はメモの節に書く。レシピに URL 専用の欄は足さない
- 取り込む前に、ファイルごとに読み込みとバリデーションを行い、結果を **インポートダイアログ (S02a)** に出す
  - 読み込めたレシピが 1 件なら、その内容をプレビューで見せてから取り込む
  - 2 件以上なら一覧で見せ、ファイルごとに取り込むかどうかを選べるようにする。エラーのあるファイルは選べない
- 取り込むレシピの名前が既存のレシピや、同時に取り込む別のファイルと同じなら警告を出す。
  警告の付いたレシピは、既定では取り込まない側に置く
- 材料の量の書き方の揺れをアプリ側で吸収する (「1/2」「1と1/2」「１．５」「大さじ2」「3個」「2〜3」「少々」「適量」など)
- 選んだレシピは 1 回の操作でまとめて登録する。途中で失敗したら 1 件も登録しない
- AI にレシピを変換させるためのプロンプトのひな形と、インポート形式の説明を `docs/recipe-import.md` に置く

### Non-goals

- **URL からの取得や AI 変換をアプリ内で行うこと**：外部通信をしない前提を崩すため扱わない。
- **CSV や JSON などほかの形式**：Markdown 1 形式に絞る。
- **既存レシピの上書きや統合**：同名のレシピは警告するだけで、取り込むなら別のレシピとして追加する。
- **エクスポート**：アプリからレシピをファイルに書き出す機能は、S09 のデータの import/export で扱う。
- **ドラッグ＆ドロップでの読み込み**：ボタンからの選択だけにする。
- **取り込む前の内容の編集**：ダイアログでは直さない。直したいときは元のファイルを直して選び直すか、取り込んだあとに S04 で直す。
- **名前の表記ゆれを重複とみなすこと**：「肉じゃが」と「肉ジャガ」は別の名前として扱う。

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `recipe`：S02 からのインポート、インポート形式、量の表記ゆれの吸収、プレビューと選択、同名の警告、まとめての登録を加える。

## Impact

### 新設ファイル

- `internal/domain/recipe/importfile.go`：`ImportFile` / `ImportCandidate` / 問題コードの型と、Markdown 1 ファイルをレシピに読む `ParseImportFile`
- `internal/domain/recipe/importfile_test.go`：`ParseImportFile` の table-driven test
- `internal/domain/recipe/amount.go`：量の表記ゆれを数値と単位に直す `ParseAmount`
- `internal/domain/recipe/amount_test.go`：`ParseAmount` の table-driven test
- `frontend/src/components/recipe/recipe-import-button.tsx`：「インポート」ボタンと、ファイル用とフォルダ用の 2 つの隠し input
- `frontend/src/components/recipe/read-import-files.ts`：選ばれた `File` から `.md` だけを選び、名前と中身の組にする
- `frontend/src/components/recipe/recipe-import-dialog.tsx`：S02a。件数に応じてプレビューと一覧を切り替え、取り込みを確定する
- `frontend/src/components/recipe/recipe-import-preview.tsx`：1 件のときのプレビュー
- `frontend/src/components/recipe/recipe-import-list.tsx`：2 件以上のときの一覧 (選択チェック、ファイル名、レシピ名、材料数、状態)
- `frontend/e2e/recipe-import.spec.ts`：インポートの E2E
- `frontend/e2e/fixtures/recipe-import/`：E2E で読み込ませる Markdown ファイル一式
- `docs/recipe-import.md`：インポート形式の説明と、AI 向けプロンプトのひな形
- `docs/wireframes/S02a_recipe_import_dialog.drawio.svg`：インポートダイアログのワイヤー (プレビューと一覧の 2 状態)

### 更新ファイル

- `internal/config/defaults.go`：`UnquantifiedWords` (量を持たない言葉) と `RecipeImportDefaultServings` を追加
- `internal/config/defaults_test.go`：上の 2 つの既定値を検証
- `internal/service/recipe.go`：`PreviewImport` と `Import` を追加。`RecipeService` に `cfg config.Config` を持たせる
- `internal/service/recipe_test.go`：`PreviewImport` と `Import` の BDD 風テストを追加
- `internal/storage/sqlite/recipe_repo.go`：複数のレシピを 1 トランザクションで登録する `CreateMany` を追加
- `internal/storage/sqlite/recipe_repo_test.go`：`CreateMany` のテストを追加
- `go.mod` / `go.sum`：`gopkg.in/yaml.v3` と `golang.org/x/text` を indirect から direct に上げる (どちらも既に依存グラフにある)
- `frontend/src/components/recipe/recipe-detail.tsx`：材料、手順、メモの欄を `RecipeContentSections` として切り出し、S03 とプレビューで共有する
- `frontend/src/routes/recipes.tsx`：ヘッダに `RecipeImportButton` を置き、`RecipeImportDialog` を開く
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：`pages.recipes.import.*` を追加
- `frontend/e2e/bindings-shim/recipeservice.ts`：`PreviewImport` と `Import` を追加
- `docs/product.md`：§1 の可変値の表に量を持たない言葉とインポート時の人数の既定値、§2 画面一覧に S02a、§4 S02 にインポートの項と S02a へのリンクを追加
- `docs/wireframes/README.md`：ファイル一覧に S02a を追加
- `openspec/specs/recipe/spec.md`：archive 時に delta を合成 (手で直接は触らない)

### 生成物 (再生成のみ)

- `frontend/bindings/`：`RecipeService.PreviewImport` / `Import` と `recipe.ImportFile` / `recipe.ImportCandidate` の型 (`wails3 generate bindings -clean=true -ts -i`)

### migration

- なし。既存の `recipes` / `recipe_ingredients` にそのまま登録する

### 前提と順序

- 進行中の `add-ingredient-name-matching` とは独立に apply できる。どちらも `recipe` の spec に Requirement を足すが、
  対象の画面が違う (向こうは S04、こちらは S02 と S02a) ので delta はぶつからない

### 後続 proposal への引き継ぎ

- S09 (設定画面)：`UnquantifiedWords` と `RecipeImportDefaultServings` を設定項目にする。S09 のデータ import/export を作るときは、
  本 proposal のインポート形式をエクスポートの出力形式にも使えるか検討する
- 名前の表記ゆれ (カタカナとひらがな、全角と半角) を重複とみなしたくなったら、`add-ingredient-name-matching` の表記をそろえる処理を流用する
