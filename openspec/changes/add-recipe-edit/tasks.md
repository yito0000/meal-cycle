## 1. Go モジュール依存追加

- [x] 1.1 `go get github.com/google/uuid@latest` を実行し `go.mod` / `go.sum` に google/uuid v1.6+ を direct として追加する
      (完了判定: `go mod tidy` 後に `go build ./` が通り、`go.mod` の `require` に `github.com/google/uuid` が indirect でない形で載る)

## 2. ドメイン層 (internal/domain/recipe 拡張)

- [x] 2.1 `internal/domain/recipe/recipe.go` を更新し、`RecipeIngredient` struct を追加する
      (フィールド: `Name string` / `Amount float64` / `Unit string`。JSON tag lowerCamelCase)
      (完了判定: `go build ./internal/...` が通る)
- [x] 2.2 `internal/domain/recipe/recipe.go` の `Recipe` struct に次のフィールドを追加する
      (`Servings int` / `Ingredients []RecipeIngredient` / `Steps string` / `Notes string` / `CreatedAt string`。
      JSON tag lowerCamelCase。既存 `IngredientCount int` は残し、doc comment に「Repository 側で `len(Ingredients)` から算出する」旨を明記)
      (完了判定: `go build ./internal/...` が通る)
- [x] 2.3 `internal/domain/recipe/recipe_test.go` を新設し、`Recipe` struct の JSON tag が期待どおりかを確認する table-driven test を追加する
    - `Recipe{Name: "肉じゃが", Servings: 2}` を Marshal して `"name":"肉じゃが"` / `"servings":2` が含まれる
    - `Ingredients` が nil の場合 JSON 出力は `"ingredients":null` であること (フロント側は `?? []` で扱う前提)
    - `RecipeIngredient{Name: "玉ねぎ", Amount: 1.5, Unit: "個"}` を Marshal して `"name":"玉ねぎ"` / `"amount":1.5` / `"unit":"個"` が含まれる
    - `Steps` が空文字の場合 `"steps":""` になること
    - (完了判定: `go test ./internal/domain/recipe/...` が pass する)

## 3. SQLite マイグレーション

- [x] 3.1 `internal/storage/sqlite/migrations/00002_recipes.sql` を新設する
    - `recipes` テーブル: `id TEXT PRIMARY KEY / name TEXT NOT NULL / servings INTEGER NOT NULL DEFAULT 1 /
      tags_json TEXT NOT NULL DEFAULT '[]' / steps TEXT NOT NULL DEFAULT '' /
      notes TEXT NOT NULL DEFAULT '' / created_at TEXT NOT NULL / updated_at TEXT NOT NULL`
    - `recipe_ingredients` テーブル: `recipe_id TEXT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE /
      name TEXT NOT NULL / amount REAL NOT NULL / unit TEXT NOT NULL /
      position INTEGER NOT NULL / PRIMARY KEY (recipe_id, position)`
    - `CREATE INDEX idx_recipes_updated_at ON recipes (updated_at DESC)`
    - Down: 2 テーブルを DROP (`recipe_ingredients` → `recipes` の順)
      (完了判定: 3.2 のテストで migration が Up/Down 両方通る)
- [x] 3.2 `internal/storage/sqlite/sqlite_test.go` の `TestOpenAndMigrate` を更新し、goose_db_version の最新値が 2 になることを確認する
      (完了判定: `go test ./internal/storage/sqlite/...` が pass する)

## 4. SQLite リポジトリ (Recipe)

- [x] 4.1 `internal/storage/sqlite/recipe_repo.go` を新設し、`RecipeRepo` struct と `New(db *sql.DB) *RecipeRepo` を定義する
      (完了判定: `go build ./internal/storage/sqlite/...` が通る)
- [x] 4.2 `RecipeRepo.List(ctx) ([]recipe.Recipe, error)` を実装する
      (recipes を updated_at DESC で全件取得、`IngredientCount` は `SELECT COUNT(*) FROM recipe_ingredients WHERE recipe_id = ?`
      のサブクエリ or LEFT JOIN で 1 クエリに合成する。`Ingredients` / `Steps` / `Notes` は空値で返す — List は summary 用途)
      (完了判定: 4.7 のテストが pass する)
- [x] 4.3 `RecipeRepo.GetByID(ctx, id string) (recipe.Recipe, error)` を実装する
      (recipes 1 行 + recipe_ingredients 全行を取得し 1 つの Recipe に詰める。tags_json は `json.Unmarshal` で `[]string` に戻す。
      steps / notes はそのまま文字列。該当なしは `ErrRecipeNotFound`)
      (完了判定: 4.7 のテストが pass する)
- [x] 4.4 `RecipeRepo.Create(ctx, r recipe.Recipe) (recipe.Recipe, error)` を実装する
      (処理: uuid.NewV7 で ID 生成 → トランザクション開始 → recipes INSERT → recipe_ingredients を position 順に一括 INSERT →
      commit。created_at / updated_at は `time.Now().UTC().Format(time.RFC3339)` を採用。バリデーションは呼び出し元 Service 側で
      済ませ、Repository は入力を信頼する。空 name のみ最終防衛線として `ErrRecipeNameEmpty` を返す)
      (完了判定: 4.7 のテストが pass する)
- [x] 4.5 `RecipeRepo.Update(ctx, id string, r recipe.Recipe) (recipe.Recipe, error)` を実装する
      (処理: 存在確認 (GetByID) → トランザクション開始 → recipes UPDATE (updated_at を現在時刻に、同秒衝突時は +1s) →
      recipe_ingredients を対象 recipe_id 分 DELETE → 新しい ingredient 群を position 順に INSERT → commit。該当 ID が無ければ
      `ErrRecipeNotFound`。name が空なら `ErrRecipeNameEmpty`)
      (完了判定: 4.7 のテストが pass する)
- [x] 4.6 `ErrRecipeNotFound` / `ErrRecipeNameEmpty` を `internal/storage/sqlite/recipe_repo.go` に定義する
      (`var ErrRecipeNotFound = errors.New("recipe not found")` 形式)
      (完了判定: 4.7 のテストが pass する)
- [x] 4.7 `internal/storage/sqlite/recipe_repo_test.go` を新設し、実 SQLite (`t.TempDir()`) での table-driven test を追加する
    - List: 空 DB で空スライスを返す
    - List: 3 件 Create したものを updated_at DESC で返す
    - List: IngredientCount が recipe_ingredients の件数と一致する
    - GetByID: Create した recipe を取り出すと Ingredients (name / amount / unit) / Steps / Notes / Tags が保存時と一致する
    - GetByID: 存在しない ID は ErrRecipeNotFound を返す
    - Create: name 空だと ErrRecipeNameEmpty を返し、DB に書き込まない (Count 前後で不変)
    - Create: Ingredients の position が入力順で保存され、GetByID で同じ順に取り出せる
    - Create: Steps に Markdown テキストを入れて往復させ、素通しで返ることを確認する
    - Update: 既存 recipe の name を変えると GetByID で新しい name が返る
    - Update: Update 後 updated_at が Create 時より新しい (同秒衝突時に +1s される挙動も含む)
    - Update: Ingredients を全部入れ替えても古い recipe_ingredients は残らない (COUNT で確認)
    - Update: 存在しない ID は ErrRecipeNotFound を返す
    - Recipe 削除経路のテストは本 proposal 対象外 (add-recipe-deletion で追加)
    - (完了判定: `go test ./internal/storage/sqlite/...` が pass する)

## 5. Wails サービス (RecipeService 差し替え)

- [x] 5.1 `recipeservice.go` を更新し、`RecipeService` struct に `repo *sqlite.RecipeRepo` を持たせる
      (`NewRecipeService(db *sql.DB) *RecipeService` に変更し、`fixtureRecipes` 変数と関連定義は削除)
      (完了判定: `go build ./` が通る)
- [x] 5.2 `RecipeService.List(ctx)` を Repository.List に委譲する
      (fixture 複製ロジックは削除。Repository が新スライスを返すため)
      (完了判定: 5.7 のテストが pass する)
- [x] 5.3 `RecipeService.GetByID(ctx, id string) (recipe.Recipe, error)` を Repository.GetByID に委譲する形で追加する
      (完了判定: 5.7 のテストが pass する)
- [x] 5.4 `RecipeService.Create(ctx, input recipe.Recipe) (recipe.Recipe, error)` を追加する
      (処理: 入力バリデーション (`validateRecipeInput`) → Repository.Create に委譲。input.ID / input.CreatedAt / input.UpdatedAt /
      input.IngredientCount は無視する旨を doc comment に明記)
      (完了判定: 5.7 のテストが pass する)
- [x] 5.5 `RecipeService.Update(ctx, id string, input recipe.Recipe) (recipe.Recipe, error)` を追加する
      (処理: 入力バリデーション → Repository.Update に委譲)
      (完了判定: 5.7 のテストが pass する)
- [x] 5.6 `recipeservice.go` に `validateRecipeInput(input recipe.Recipe) error` を追加する
    - name: trim 後 1〜200 字。違反時 `ErrRecipeNameEmpty` / `ErrRecipeNameTooLong` を返す
    - servings: 1〜99 の整数。違反時 `ErrRecipeServingsOutOfRange` を返す
    - Ingredients: 各行の Name (trim 後) が空なら `ErrRecipeIngredientNameEmpty` を返す (Amount / Unit は空可)
    - Steps: 4000 字以内。違反時 `ErrRecipeStepsTooLong` を返す
    - Notes: 2000 字以内。違反時 `ErrRecipeNotesTooLong` を返す
    - Tags: 各要素 trim 後 30 字以内。違反時 `ErrRecipeTagTooLong` を返す。空タグは呼び出し側 (フロント) で捨てる想定
    - エラー変数群は `recipeservice.go` に `var ErrRecipeNameTooLong = errors.New("recipe name too long")` 形式で定義
      (完了判定: 5.7 のテストが pass する)
- [x] 5.7 `recipeservice_test.go` を全面書き換えし、実 SQLite (`t.TempDir()`) 上で BDD 風テストを追加する
    - should return an empty list when the database is fresh
    - should return recipes ordered by updatedAt desc when multiple exist
    - should create a recipe and make it visible in List when Create succeeds
    - should return the created recipe with a non-empty ID and updatedAt
    - should return ErrRecipeNameEmpty when Create is called with an empty name
    - should return ErrRecipeNameTooLong when Create is called with a 201-character name
    - should return ErrRecipeServingsOutOfRange when Create is called with servings=0
    - should return ErrRecipeServingsOutOfRange when Create is called with servings=100
    - should return ErrRecipeIngredientNameEmpty when an ingredient row has empty Name
    - should return ErrRecipeStepsTooLong when Steps exceeds 4000 characters
    - should return ErrRecipeNotesTooLong when Notes exceeds 2000 characters
    - should return ErrRecipeTagTooLong when a single tag exceeds 30 characters
    - should update an existing recipe when Update is called with valid input
    - should refresh updatedAt when Update is called
    - should replace ingredients when Update is called with a different ingredient set
    - should return ErrRecipeNotFound when Update is called with an unknown id
    - should return the recipe when GetByID is called with an existing id
    - should return ErrRecipeNotFound when GetByID is called with an unknown id
    - should preserve Markdown text in Steps and Notes across Create → GetByID
    - (完了判定: `go test ./` が pass する)

## 6. main.go 更新

- [x] 6.1 `main.go` を更新し、`NewRecipeService()` を `NewRecipeService(db)` に変更する
      (IngredientService の登録は行わない — 本 proposal では ingredient capability を新設しないため)
      (完了判定: `go build ./` が通り、`task dev` で起動して devtools console から `RecipeService.List()` が配列を返すことを目視確認する)

## 7. Wails バインディング再生成

- [x] 7.1 `wails3 generate bindings` を実行する
      (完了判定: `frontend/bindings/github.com/yito0000/meal-cycle/` の `recipeservice.js` に `Create` / `Update` / `GetByID` が
      追加され、`internal/domain/recipe/models.js` の `Recipe` に新規フィールド `Servings` / `Ingredients` / `Steps` / `Notes` /
      `CreatedAt` と `RecipeIngredient` (`Name` / `Amount` / `Unit`) が反映されていることを目視で確認する。ingredientservice.js
      は生成されない (Service 未登録のため))

## 8. 可変値定数追加

- [x] 8.1 `internal/config/defaults.go` の `Config` struct に `RecipeListHighlightMs int` フィールド (JSON tag `recipeListHighlightMs`) を追加する
      (完了判定: `go build ./internal/...` が通る)
- [x] 8.2 `Defaults()` で `RecipeListHighlightMs: 2000` を返すよう更新する
      (2000ms = 2 秒。保存後ハイライト表示時間)
      (完了判定: 8.3 のテストが pass する)
- [x] 8.3 `internal/config/defaults_test.go` を更新し、`RecipeListHighlightMs == 2000` を検証するテストケースを追加する
    - Defaults().RecipeListHighlightMs は 2000 を返す
    - (完了判定: `go test ./internal/config/...` が pass する)

## 9. フロントエンド shadcn プリミティブ追加

- [x] 9.1 `npm install @radix-ui/react-dialog zod fast-equals` を frontend ディレクトリで実行する
      (完了判定: `frontend/package.json` に 3 パッケージが追加され、`cd frontend && npm run type-check` が通る)
- [x] 9.2 shadcn/ui docs (https://ui.shadcn.com/docs/components/dialog) から `Dialog` の TSX を
      `frontend/src/components/ui/dialog.tsx` に手動コピーで新設する (CLI は使わない)
      (完了判定: ファイルが存在し、`cd frontend && npm run type-check` が通る)
- [x] 9.3 shadcn/ui docs から `Label` / `Textarea` の TSX を `frontend/src/components/ui/` にそれぞれ手動コピーで新設する
      (既に導入済みのプリミティブはスキップ。2 ファイルとも小さいため 1 タスクにまとめる)
      (完了判定: 2 ファイルが揃い、`cd frontend && npm run type-check` が通る)

## 10. フロントエンド (S04 サブコンポーネント)

- [x] 10.1 `frontend/src/components/recipe/use-recipe-form.ts` を新設し、`useRecipeForm(initial)` フックを実装する
      (state: name / servings / tags / ingredients / steps / notes。`initial` を保持し、`fast-equals` の `deepEqual` で
      dirty 判定を提供 (`isDirty` bool)。値更新のセッター群と `reset(initial)` を export)
      (完了判定: `cd frontend && npm run type-check` が通る。挙動は 13.x で目視確認)
- [x] 10.2 `frontend/src/components/recipe/recipe-ingredient-row.tsx` を新設する
      (props: 材料 1 行 (`name / amount / unit`) と変更ハンドラ / 削除ハンドラ。名前用 text input + 量用 number input + 単位用 text input
      + 削除ボタン (`×`)。食材コンボボックスは使わない)
      (完了判定: 13.x で目視確認)
- [x] 10.3 `frontend/src/components/recipe/recipe-tags-input.tsx` を新設する
      (props: `value: string[]` / `onChange(next: string[])`。入力用テキストボックス + 追加ボタン (`＋`) + 現在の tags を chip として
      横並び表示 + 各 chip に `×` ボタン。Enter キーでも確定。空文字は捨てる。30 字を超える入力は input 側で `maxLength=30` で防ぐ)
      (完了判定: 13.x で目視確認)
- [x] 10.4 `frontend/src/components/recipe/unsaved-changes-dialog.tsx` を新設する
      (props: `open` / `onDiscard` / `onKeepEditing`。shadcn Dialog を薄くラップし、i18n key `pages.recipes.edit.unsaved_confirm.*` を使用)
      (完了判定: `cd frontend && npm run type-check` が通る)

## 11. フロントエンド (S04 レシピ登録/編集画面本体)

- [x] 11.1 `frontend/src/components/recipe/recipe-edit-form.tsx` を新設する
      (props: `mode: "new" | "edit"` / `initialRecipe: Recipe | null` / `onSave(input): Promise<Recipe>` / `onCancel(): void`。
      内部で `useRecipeForm(initialRecipe ?? blank)` を使い、上部固定バー + 左カラム (基本情報 + タグ chip 入力 + 材料表) + 右カラム
      (手順 textarea + メモ textarea) を組む。手順 textarea は `maxLength=4000`、その下に「残り N 字」表示と「Markdown 記法で書けます
      (プレビューは S03 で追加予定)」ヒントを出す。メモ textarea は `maxLength=2000`。保存ボタン押下で入力バリデーションを行い、
      違反ならフィールド近傍にエラー表示。全部通れば onSave を呼び、失敗時は上部にエラーバナー。dirty 状態でキャンセルを押すと確認ダイアログ)
      (完了判定: 13.x で目視確認)
- [x] 11.2 `frontend/src/components/recipe/recipe-edit-form.tsx` にフロント側バリデーションを実装する
    - name: trim 後 1〜200 字。違反時 `edit.validation.name_required` / `save_error.name_too_long`
    - servings: 1〜99。違反時 `edit.save_error.servings_out_of_range`
    - Ingredients: 各行 name trim 後 1 字以上。違反時 `edit.save_error.ingredient_name_required` を該当行に出す
    - Steps: 4000 字以内 (超過分は `maxLength` で入力段階で防ぐ)。保存時にも冗長チェック
    - Notes: 2000 字以内
    - Tags: 各要素 trim 後 30 字以内 (入力時 `maxLength=30` で防ぐ + 保存時チェック)
    - Go 側から返るエラーコードを i18n key にマップする helper を用意する
      (完了判定: 13.x で目視確認)
- [x] 11.3 `frontend/src/routes/recipes/new.tsx` を新設する
      (`RecipeService.Create` を呼び、成功時に `router.navigate({ to: "/recipes", search: { highlight: created.id } })`。
      失敗時は Go 側エラーコードに応じて i18n key を選ぶ。`RecipeEditForm` に `mode="new"` と `initialRecipe={null}` を渡す。
      `Recipe` は `import type` で取り込み、`const draft: Recipe = {...}` のリテラルで組む (TS モードバインディングは interface のため))
      (完了判定: 13.x で目視確認)
- [x] 11.4 `frontend/src/routes/recipes/$recipeId.edit.tsx` を新設する
      (`RecipeService.GetByID(recipeId)` を useQuery で読み、ロード中は placeholder、失敗時はエラーメッセージ + 一覧に戻るリンク。
      成功時に `RecipeEditForm` に `mode="edit"` と `initialRecipe={data}` を渡す。onSave は `RecipeService.Update(recipeId, input)`。
      `Recipe` は `import type`、リテラル組み)
      (完了判定: 13.x で目視確認)

## 12. フロントエンド (S02 更新 + router + i18n)

- [x] 12.1 `frontend/src/router.tsx` を更新し、`/recipes/new` と `/recipes/$recipeId/edit` のルートを追加する
      (`recipesRoute` の兄弟として展開。`/recipes` はハイライト ID を search param で受け取れるよう `validateSearch` に
      zod スキーマ (`z.object({ highlight: z.string().optional() })`) を追加)
      (完了判定: `cd frontend && npm run type-check` が通り、`cd frontend && npm run build:dev` が通る)
- [x] 12.2 `frontend/src/components/recipe/recipe-table.tsx` を更新し、行 `onClick` prop と `highlightId?: string` prop を受け取れるようにする
      (行の onClick で受け取ったコールバックを呼ぶ。highlightId と一致する行に Tailwind の `bg-accent` を当ててテーマと揃える)
      (完了判定: `cd frontend && npm run type-check` が通る。目視は 13.x で確認)
- [x] 12.3 `frontend/src/routes/recipes.tsx` を更新し、「＋ 新しいレシピ」ボタンの `onClick` を
      `router.navigate({ to: "/recipes/new" })` に変更する
      (完了判定: `cd frontend && npm run type-check` が通る。目視は 13.x で確認)
- [x] 12.4 `frontend/src/routes/recipes.tsx` を更新し、`useSearch` で highlight 値を受け取り `RecipeTable` に流す。
      `useEffect` で `RecipeListHighlightMs` (defaults 経由でフロントに露出する仕組みが未整備なら本タスクではハードコード 2000 で
      仮置きし、S09 実装 proposal で config service 化する旨を TODO コメントに残す) 経過後に自動クリア (`router.navigate` で search を空に更新)
      (完了判定: 目視で新規保存後 2 秒間ハイライトされることを確認 — 13.x)
- [x] 12.5 `frontend/src/routes/recipes.tsx` を更新し、`RecipeTable` の `onRowClick` に
      `(id) => router.navigate({ to: "/recipes/$recipeId/edit", params: { recipeId: id } })` を渡す
      (完了判定: 目視で行クリックで S04 編集画面に飛ぶことを確認 — 13.x)
- [x] 12.6 `frontend/src/locales/ja.json` の `pages.recipes` を拡張する
    - `edit.title_new` = "新しいレシピ"
    - `edit.title_edit` = "レシピを編集: {{name}}"
    - `edit.save` = "保存"
    - `edit.cancel` = "キャンセル"
    - `edit.label.name` / `label.servings` / `label.tags` / `label.ingredients` / `label.steps` / `label.notes` /
      `label.amount` / `label.unit` = "名前" / "人数" / "タグ" / "材料" / "手順" / "メモ" / "量" / "単位"
    - `edit.add_ingredient` = "＋ 材料を追加"
    - `edit.add_tag` = "＋ タグを追加"
    - `edit.remove_ingredient` = "材料行を削除"
    - `edit.remove_tag` = "タグを削除"
    - `edit.tag_placeholder` = "タグを入力して Enter"
    - `edit.markdown_hint` = "Markdown 記法で書けます (プレビューは S03 で追加予定)"
    - `edit.chars_remaining` = "残り {{count}} 字"
    - `edit.unsaved_confirm.title` = "未保存の変更を破棄しますか"
    - `edit.unsaved_confirm.body` = "編集した内容は保存されません"
    - `edit.unsaved_confirm.discard` = "破棄"
    - `edit.unsaved_confirm.keep_editing` = "編集を続ける"
    - `edit.save_error.generic` = "保存に失敗しました"
    - `edit.save_error.name_empty` = "レシピ名を入力してください"
    - `edit.save_error.name_too_long` = "レシピ名は 200 字以内で入力してください"
    - `edit.save_error.servings_out_of_range` = "人数は 1〜99 で入力してください"
    - `edit.save_error.ingredient_name_required` = "材料名を入力してください"
    - `edit.save_error.steps_too_long` = "手順は 4000 字以内で入力してください"
    - `edit.save_error.notes_too_long` = "メモは 2000 字以内で入力してください"
    - `edit.save_error.tag_too_long` = "タグは 30 字以内で入力してください"
    - `edit.validation.name_required` = "レシピ名は必須です"
    - `edit.loading` = "読み込み中..."
    - `edit.load_error` = "レシピの読み込みに失敗しました"
    - `edit.back_to_list` = "一覧に戻る"
      (完了判定: 目視で該当画面が日本語で表示される — 13.x)
- [x] 12.7 `frontend/src/locales/en.json` に 12.6 と同キーの英訳を追加する
      (完了判定: 12.6 と同時に検証)

## 13. 動作確認とテスト実行

- [ ] 13.1 `task dev` で起動し、S02 「＋ 新しいレシピ」ボタンから S04 新規登録画面に遷移することを目視確認する
- [ ] 13.2 S04 新規登録画面で名前だけ入力して保存し、S02 に戻って追加行がハイライトされ、2 秒後にハイライトが消えることを目視確認する
- [ ] 13.3 S04 新規登録画面で材料行に「玉ねぎ / 1 / 個」を入力して保存し、S02 に戻ってから同レシピを開いて材料が保存されていることを目視確認する
- [ ] 13.4 S02 で追加した行をクリックすると S04 編集画面に遷移し、既存値がロードされることを目視確認する
- [ ] 13.5 S04 編集画面で人数を変更して保存し、S02 に戻って更新日が変わり (行の並び順が変わる)、ハイライトされることを目視確認する
- [ ] 13.6 S04 でタグ入力欄に「和食」を入力して `＋` (または Enter)、続けて「煮物」を追加し、`×` で「和食」を削除できることを目視確認する
- [ ] 13.7 S04 で名前を入力してから「キャンセル」を押し、確認ダイアログが出て「編集を続ける」で戻り、「破棄」で S02 に戻ることを目視確認する
- [ ] 13.8 S04 で名前を空のまま保存を押し、名前欄にエラー表示が出て S04 に留まることを目視確認する
- [ ] 13.9 S04 で名前欄に 201 字以上を入れて保存を押し、上限超のエラー表示が出て S04 に留まることを目視確認する
- [ ] 13.10 S04 で人数欄に 0 や 100 を入れて保存を押し、範囲エラー表示が出て S04 に留まることを目視確認する
- [ ] 13.11 S04 で材料行を追加して名前欄を空のまま保存を押し、その行にエラー表示が出て S04 に留まることを目視確認する
- [ ] 13.12 S04 で手順欄に Markdown (`## 見出し` や `- リスト`) を入力して保存し、再度開いた際にテキストが素通しで表示されることを目視確認する
       (プレビューは出ないことも確認)
- [ ] 13.13 S04 で手順欄の残り字数表示がリアルタイムに減ることを目視確認する
- [ ] 13.14 アプリを完全に終了して再起動し、追加・編集したレシピが引き続き S02 に表示されることを目視確認する
- [x] 13.15 `go test ./internal/...` を実行し、全テストが pass することを確認する
- [x] 13.16 `go test ./` を実行し、サービス層のテストが pass することを確認する
- [x] 13.17 `cd frontend && npm run type-check` を実行し、TypeScript 型エラーが無いことを確認する
- [x] 13.18 `cd frontend && npm run build:dev` を実行し、Vite ビルドが通ることを確認する
- [x] 13.19 `task build` を実行し、TS モードバインディングでもプロダクションビルドが通ることを確認する
- [x] 13.20 `task lint` を実行し、Go / Frontend の lint が通ることを確認する
