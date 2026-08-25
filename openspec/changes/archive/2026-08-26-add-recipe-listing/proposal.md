## Why

MVP の第 2 画面である S02 レシピ一覧を、まず「登録済みレシピをテーブルで一覧する」ところまで通し、以降のレシピ詳細ドロワー
(S03) / 登録編集 (S04) / SQLite 永続化 proposal の土台にする。実データ層 (repository + migration) はまだ整っていないため、
本 proposal では **Wails サービスが固定値を返す**ところで打ち止めにし、UI + サービス境界 + ドメイン型 を先に確定させる。
これにより、次の `add-recipe-crud` は「サービスの中身を repo 実装に差し替えるだけ」で進められるようにする。

## What Changes

- Wails サービス `RecipeService` を新設し、`List(ctx)` メソッドで固定値の `[]recipe.Recipe` を返す
- ドメインパッケージ `internal/domain/recipe/` を新設し、`Recipe` 型 (ID / Name / Tags / IngredientCount / UpdatedAt) を定義する
  (`IngredientCount` は本 proposal の一覧では非表示だが、後続 proposal の詳細ドロワー / 献立割当で使うため型に含めておく)
- `frontend/src/routes/recipes.tsx` の placeholder を実 UI に置き換え、レシピ一覧テーブルを描画する
- テーブル列は「名前 / タグ / 更新日」の 3 列とし、行内は 1 レシピにつき 1 行で表示する
- 検索欄 (レシピ名の部分一致) を画面上部に配置し、フロント側でクライアントフィルタする (サービス呼び出しは 1 回のみ)
- ソート切替 (「更新日 降順」= デフォルト / 「名前 昇順」) をヘッダ横に配置し、クライアント側で並び替える
  (プルダウンは shadcn/ui の `Select` プリミティブを導入して WebView 内で描画し、Wails の Zoom 設定に追従させる)
- 画面右上に「＋ 新しいレシピ」ボタンを配置する。**本 proposal ではクリックしても何も起きない (後続 proposal で遷移を実装)**
- i18n キーを `frontend/src/locales/{ja,en}.json` の `pages.recipes.*` に追加。日本語 UI 文字列はすべて `t()` 経由

### Non-goals

- **永続化しない**: SQLite テーブル / migration / repository は本 proposal では作らない
  (次以降の proposal でサービス実装を差し替える前提)
- **行クリック → S03 レシピ詳細ドロワー**: ワイヤーで想定されている挙動だが、本 proposal では行を非活性表示にとどめ、
  ドロワーは `add-recipe-detail-drawer` (別 proposal) で追加する
- **「＋ 新しいレシピ」ボタンの遷移先**: ボタンは表示するがクリック時の遷移は実装しない。S04 画面自体が `add-recipe-edit`
  (別 proposal) の対象のため、そこで遷移を有効化する
- **レシピの新規作成 / 編集 / 削除**: `add-recipe-edit` / `add-recipe-crud` の担当
- **タグ・材料での絞り込み**: MVP では検索欄はレシピ名の部分一致のみ。タグ絞り込みは将来 S10 (`add-recipe-search`) で追加
- **他画面 (S03/S04 等)**: サイドバー他項目のスタブは既存のまま。実体は各 capability の proposal で足す

## Capabilities

### New Capabilities

- `recipe`: レシピの一覧・詳細・登録編集を扱う capability。
  本 proposal で追加する要件は「一覧表示」「レシピ名検索」「ソート切替」「新規レシピボタンの配置」の 4 要件。

### Modified Capabilities

なし (recipe は本 proposal で新設)。

## Impact

- **コード (新設ファイル)**:
  - `internal/domain/recipe/recipe.go` — `Recipe` struct (ID / Name / Tags / IngredientCount / UpdatedAt)
  - `internal/domain/recipe/recipe_test.go` — table-driven でドメイン層のヘルパ (追加する場合のみ) を検証
  - `recipeservice.go` (package main) — `RecipeService` struct と `List(ctx)` メソッド
  - `recipeservice_test.go` (package main) — BDD 風テスト (固定値の返却検証)
  - `frontend/src/components/recipe/recipe-table.tsx` — レシピテーブル本体 (列描画・行描画・空状態)
  - `frontend/src/components/recipe/recipe-search-input.tsx` — 検索欄 (制御コンポーネント)
  - `frontend/src/components/recipe/recipe-sort-select.tsx` — ソート切替 (「更新日 降順」/「名前 昇順」)
  - `frontend/src/components/recipe/recipe-sort.ts` — ソート key 定数・比較関数
  - `frontend/src/components/ui/select.tsx` — shadcn/ui の Select プリミティブ (Radix ベース、手動コピペ)
- **コード (更新ファイル)**:
  - `main.go` — `application.Options.Services` に `RecipeService` を追加登録
  - `frontend/src/routes/recipes.tsx` — placeholder を実 UI に置換え、TanStack Query で `RecipeService.List` をラップ
  - `frontend/src/locales/ja.json` — `pages.recipes.*` を検索欄 / ソート / ボタン / エラー文言等に拡張、placeholder キーは削除
  - `frontend/src/locales/en.json` — 同キーの英訳を追加
- **生成物 (再生成のみ)**: `frontend/bindings/` を `wails3 generate bindings` で更新 (`recipeservice.js` と
  `internal/domain/recipe/models.js` が生成される。git 管理外)
- **migration**: 追加なし (本 proposal では SQLite に触らない)
- **依存追加**: `@radix-ui/react-select` (フロントエンド) 1 個。shadcn/ui の Select プリミティブが依存するため。
  Go モジュールの追加は無し
- **テスト**: Go 側は service (固定値応答) の単体テスト。ドメイン層はヘルパを持たないため struct のフィールド定義のみで
  テストは書かない (追加するなら table-driven で書く方針)。Frontend 側は runner 未導入のため type-check + build のみ

### 後続 proposal への引き継ぎ

- `RecipeService.List` の**中身**を repository 呼び出しに差し替える (`add-recipe-crud` の一部として実施)。
  シグネチャと戻り値型は本 proposal で確定させておく
- SQLite の `recipes` テーブル + migration + `internal/storage/sqlite/recipe_repo.go` は `add-recipe-crud` で新設
- 行クリック → S03 レシピ詳細ドロワーは `add-recipe-detail-drawer` で追加
- 「＋ 新しいレシピ」ボタンの遷移 (S04) は `add-recipe-edit` で追加
- タグ / 材料絞り込み・全文検索は将来 S10 の `add-recipe-search` で追加
- `Recipe` 構造体には後続 proposal で `Ingredients` / `Steps` / `Notes` 等のフィールドが追加される想定。本 proposal では
  一覧に必要な最小フィールドのみを定義する
