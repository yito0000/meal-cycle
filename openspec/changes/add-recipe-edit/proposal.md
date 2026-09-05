## Why

MVP の第 3 スライスとして S04 レシピ登録/編集画面を通す。前 proposal `add-recipe-listing` が固定値で残した RecipeService の実装を SQLite 実体に差し替えつつ、
S02 の「＋ 新しいレシピ」ボタンと行クリックから S04 に遷移して、レシピの新規作成と再編集ができるようにする。これにより
「レシピを増やせる」= MVP のレシピ機能の中核 (create / update / persist) が end-to-end で成立する。

材料は本 proposal 時点では **食材マスタを持たない自由テキスト入力** とする。冷蔵庫在庫との名寄せが必要になった段階で別 proposal で
`ingredient` capability を立てる (本 proposal では扱わない)。手順とメモは Markdown 記法で書けるようにし、読み手側の描画は S03 詳細
ドロワー proposal で追加する。

## What Changes

- `RecipeService` に `Create(ctx, input) (Recipe, error)` / `Update(ctx, id, input) (Recipe, error)` / `GetByID(ctx, id) (Recipe, error)` を追加し、
  既存 `List(ctx)` の中身を固定値から SQLite 実装に差し替える。
- ドメインパッケージ `internal/domain/recipe/` に `RecipeIngredient` (Name / Amount / Unit) を追加し、`Recipe` に `Servings int` /
  `Ingredients []RecipeIngredient` / `Steps string` / `Notes string` を追加する。既存 `IngredientCount` は Repository 側で `len(Ingredients)`
  から算出する。
- SQLite マイグレーション `00002_recipes.sql` を追加し、`recipes` / `recipe_ingredients` の 2 テーブルを作る。`recipe_ingredients` は名前を
  直持ち (`name TEXT`)、食材マスタへの FK は張らない。
- `internal/storage/sqlite/recipe_repo.go` を新設し、ListAll / Create / GetByID / Update を実装する。
- **BREAKING**：`RecipeService.List` の返却データが固定 fixture から DB 実データに変わる。空 DB から起動すると 0 件で表示される
  (fixture への依存を残すテストは書き換える)。
- `frontend/src/routes/recipes/new.tsx` と `frontend/src/routes/recipes/$recipeId.edit.tsx` を新設し、S04 レシピ登録/編集画面のフル遷移ルートを追加する。
- S02 レシピ一覧の「＋ 新しいレシピ」ボタン `onClick` を `/recipes/new` への navigate に差し替える。
- S02 レシピ一覧の**行クリック**を有効化し、`/recipes/:recipeId/edit` に遷移する暫定導線を追加する
  (S03 詳細ドロワー proposal `add-recipe-detail-drawer` が入った段階で「行クリック → S03 → 編集ボタン → S04」に置換する)。
- S04 画面本体は、上部固定バー (保存 / キャンセル) + 左カラム (基本情報 + 材料表) + 右カラム (手順 + メモ) の 2 カラム構成
  (ワイヤー `docs/wireframes/S04_recipe_edit.drawio.svg` に従う)。
- 材料行は「名前 / 量 / 単位」の 3 テキスト input と削除ボタンだけの単純な行にする (食材コンボボックスは持たない)。
- 手順とメモは単一 `textarea` で Markdown 記法として入力する (レンダリングは S03 で追加)。文字数上限 (手順 4000 / メモ 2000 / 手順下部に残字数表示)
  を付ける。
- タグは chip 入力 (テキストボックスに文字入力 → Enter または `＋` ボタンで確定、`×` で削除)。カンマ区切りテキストは使わない。
- バリデーションを追加：レシピ名 1〜200 字、人数 1〜99、材料は 1 行につき名前必須 (量 / 単位は空可)、手順 4000 字、メモ 2000 字、タグ 1 個 30 字。
  フロント (`RecipeEditForm`) と Go 側 (`RecipeService` / `RecipeRepo`) の両方で検証する。Go 側は型付きエラーコードで返し、フロントで i18n key に対応付ける。
- 保存後は呼び出し元 (S02) に戻り、対象レシピをハイライトする (URL search param)。
- 未保存で「キャンセル」を押した場合は確認ダイアログを表示する。
- i18n キー `pages.recipes.edit.*` / `pages.recipes.new_button` (既存) を追加拡張する。日本語ハードコードは禁止。

### Non-goals

- **食材マスタ (`ingredient` capability)**：本 proposal では立てない。材料は自由テキスト入力とする。冷蔵庫在庫や献立との名寄せが必要になった段階で
  別 proposal `add-ingredient-master` を立てる (autocomplete / 正規化 / 単位変換の議論もそこで扱う)。
- **手順 / メモの Markdown レンダリング**：書き手側は素の `textarea` のみ。プレビューと読み手側の描画は S03 詳細ドロワー proposal
  `add-recipe-detail-drawer` で `react-markdown` などを検討する。
- **S03 レシピ詳細ドロワー**：別 proposal `add-recipe-detail-drawer`。本 proposal では S02 → S04 の暫定直結導線のみ実装する。
- **レシピの削除**：別 proposal `add-recipe-deletion` (S03 詳細ドロワー完了後を想定)。
- **タグの正規化 / タグマスタ管理 / 既存タグの候補提示**：タグは引き続き `[]string` で recipe に直接持たせる。候補提示は次回以降の別 proposal で追加する。
- **単位変換 (g ⇔ 個)**：保存された単位をそのまま扱う。
- **材料行のドラッグ並び替え**：MVP は上下ボタン (`↑` / `↓`) のみ。
- **手順の画像添付 / タイマー / チェックリスト化**：単一 Markdown テキストのみ。
- **献立側からの参照整合**：現状 `MealPlan` は `RecipeName` を直接持つ (add-mealplan-week-view の実装)。RecipeID 参照への切替は別 proposal で扱う。
- **Frontend テスト runner の導入**：Vitest 未導入。フロントの検証は `npm run type-check` + `npm run build:dev` + 手動目視で行う。

## Capabilities

### New Capabilities

(なし。ingredient capability の新設は取りやめ、`recipe` capability の Modified のみで扱う。)

### Modified Capabilities

- `recipe`：既存の「レシピ一覧 / 検索 / ソート / 新規レシピボタンの配置」に加えて、S04 レシピ登録/編集画面と、SQLite 永続化、
  「新しいレシピ」ボタンの遷移、行クリックからの編集導線、入力バリデーションを追加する。既存の
  「押しても画面遷移は起きない」要件 (add-recipe-listing) は本 proposal で置き換える。

## Impact

### 新設ファイル

- `internal/storage/sqlite/migrations/00002_recipes.sql`：`recipes` / `recipe_ingredients` の 2 テーブル追加
- `internal/storage/sqlite/recipe_repo.go`：`RecipeRepo` (List / GetByID / Create / Update)。トランザクション内で recipes + recipe_ingredients を一括保存
- `internal/storage/sqlite/recipe_repo_test.go`：実 SQLite での CRUD + 更新時の材料行入替テスト + バリデーションテスト
- `frontend/src/routes/recipes/new.tsx`：S04 新規モード ルート
- `frontend/src/routes/recipes/$recipeId.edit.tsx`：S04 編集モード ルート
- `frontend/src/components/recipe/recipe-edit-form.tsx`：S04 フォーム本体 (新規 / 編集で共有)
- `frontend/src/components/recipe/recipe-ingredient-row.tsx`：材料 1 行 (名前 / 量 / 単位 のテキスト input × 3 + 削除ボタン)
- `frontend/src/components/recipe/recipe-tags-input.tsx`：タグ chip 入力 (Enter / `＋` で確定、`×` で削除)
- `frontend/src/components/recipe/use-recipe-form.ts`：フォーム状態管理 (dirty 判定用)
- `frontend/src/components/recipe/unsaved-changes-dialog.tsx`：キャンセル確認ダイアログ
- `frontend/src/components/ui/dialog.tsx`：shadcn Dialog プリミティブ (未導入なので手動コピー)
- `frontend/src/components/ui/label.tsx`：shadcn Label プリミティブ (フォームで使用、未導入なら追加)
- `frontend/src/components/ui/textarea.tsx`：shadcn Textarea プリミティブ (手順とメモで使用、未導入なら追加)

### 更新ファイル

- `internal/domain/recipe/recipe.go`：`Recipe` struct を拡張 (`Servings int` / `Ingredients []RecipeIngredient` / `Steps string` /
  `Notes string` / `CreatedAt string` を追加)。`RecipeIngredient` (`Name string` / `Amount float64` / `Unit string`) を新設。
  `IngredientCount` は残しつつ Repository 側で `len(Ingredients)` から算出する運用に変える
- `recipeservice.go`：`fixtureRecipes` を削除。`RecipeService` に `db *sql.DB` を持たせて RecipeRepo 経由に差し替え。
  `Create` / `Update` / `GetByID` を追加。`NewRecipeService(db)` に変更。入力バリデーションをここで行い、型付きエラーで返す
- `recipeservice_test.go`：fixture 前提テストを削除し、`t.TempDir()` の実 SQLite に対する BDD 風テスト (バリデーション観点含む) に書き換え
- `main.go`：`NewRecipeService()` を `NewRecipeService(db)` に変更 (IngredientService の登録は行わない)
- `internal/config/defaults.go`：`RecipeListHighlightMs int` を追加
- `internal/config/defaults_test.go`：`RecipeListHighlightMs == 2000` の検証を追加
- `internal/storage/sqlite/sqlite_test.go`：goose_db_version の最新値が 2 になることを確認
- `frontend/src/router.tsx`：`/recipes/new` と `/recipes/$recipeId/edit` の 2 ルートを追加、`/recipes` の `validateSearch` に highlight を追加
- `frontend/src/routes/recipes.tsx`：「＋ 新しいレシピ」ボタンの `onClick` を `router.navigate("/recipes/new")` に、
  行クリック (`onRowClick`) を `router.navigate("/recipes/" + id + "/edit")` に差し替え。保存後ハイライト用の `highlightId` を URL search param から受け取り表示
- `frontend/src/components/recipe/recipe-table.tsx`：行 `onClick` を props で受け取れるようにし、ハイライト行 (`highlightId`) の背景色を切り替える
- `frontend/src/locales/ja.json`：`pages.recipes.edit.*` (title_new / title_edit / save / cancel / label.name / label.servings /
  label.tags / label.ingredients / label.steps / label.notes / add_ingredient / add_tag / markdown_hint / unsaved_confirm.\* /
  save_error / validation.\*) を追加
- `frontend/src/locales/en.json`：同キーの英訳を追加

### 生成物 (再生成のみ)

- `frontend/bindings/` を `wails3 generate bindings` で更新 (`recipeservice.js` に `Create` / `Update` / `GetByID` が追加され、
  `internal/domain/recipe/models.js` の `Recipe` に新規フィールド `Servings` / `Ingredients` / `Steps` / `Notes` /
  `RecipeIngredient` が反映される。git 管理外)

### migration

- `00002_recipes.sql` を追加:
  - `recipes (id TEXT PK, name TEXT NOT NULL, servings INTEGER NOT NULL DEFAULT 1, tags_json TEXT NOT NULL DEFAULT '[]',
    steps TEXT NOT NULL DEFAULT '', notes TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`
  - `recipe_ingredients (recipe_id TEXT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    name TEXT NOT NULL, amount REAL NOT NULL, unit TEXT NOT NULL,
    position INTEGER NOT NULL, PRIMARY KEY (recipe_id, position))`
  - `INDEX idx_recipes_updated_at ON recipes (updated_at DESC)`：S02 のデフォルト ORDER BY を効率化
- Down は 2 テーブルを DROP。理由と最終形は design.md D2 で決定する

### 依存追加

- `@radix-ui/react-dialog`：shadcn Dialog が Radix ベースのため
- `zod`：TanStack Router の `validateSearch` スキーマ用
- `fast-equals`：フォーム dirty 判定の deep compare 用
- Go 側モジュール追加：`google/uuid` (Recipe ID 発行に UUID v7 を使う)

### 後続 proposal への引き継ぎ

- **add-recipe-detail-drawer**：行クリックの遷移先を `/recipes/:id/edit` から S03 詳細ドロワー展開に置換する。ドロワー内「編集」ボタンから
  S04 に飛ぶ導線に一本化する。本 proposal で追加した「行クリック → 編集」要件は同 proposal で MODIFIED / REMOVED される。
  手順 / メモの Markdown レンダリング (`react-markdown` 等) もここで導入する
- **add-recipe-deletion**：S03 詳細ドロワー完了後に削除ボタンを足す。RecipeRepo.Delete と recipe_ingredients の
  ON DELETE CASCADE 動作は本 proposal のテーブル定義で下地を作る
- **add-mealplan-recipe-reference**：MealPlan.Entry の `RecipeName` を `RecipeID` に置き換え、S01 の表示で
  RecipeService.GetByID を引くようにする (現状 MealPlan は Recipe と疎結合)
- **add-ingredient-master**：冷蔵庫在庫や買い物リストとの名寄せが必要になった段階で `ingredient` capability を新設する。
  レシピ材料の name TEXT を ingredient テーブルへの FK に切り替えるか、突合を実行時マッチで済ませるかは、その proposal の
  design で決める。名寄せ辞書 / autocomplete / 単位変換もそこで扱う
- **add-recipe-tags-autocomplete**：既存タグからの候補提示を chip 入力に足す。SELECT DISTINCT かフロント集約かは
  そこで決める
