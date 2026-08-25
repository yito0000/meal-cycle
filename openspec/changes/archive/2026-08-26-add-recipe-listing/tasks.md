## 1. ドメイン層 (internal/domain/recipe)

- [x] 1.1 `internal/domain/recipe/recipe.go` を新設し、`Recipe` struct を定義する
      (フィールド: `ID string` / `Name string` / `Tags []string` / `IngredientCount int` / `UpdatedAt string` (YYYY-MM-DD))
      (完了判定: `go build ./internal/...` が通る)

## 2. Wails サービス (RecipeService)

- [x] 2.1 ルート pkg に `recipeservice.go` を新設し、`RecipeService` struct を定義する
      (完了判定: `go build ./` が通る)
- [x] 2.2 `recipeservice.go` に `var fixtureRecipes = []recipe.Recipe{...}` を追加し、5〜6 件の固定値を持たせる
      (少なくとも 1 件はタグ 2 個以上、少なくとも 1 件はタグ 0 個。日付は固定文字列 `"2026-08-20"` 等)
      (完了判定: 下記 2.4 のテストが pass する)
- [x] 2.3 `List(ctx context.Context) ([]recipe.Recipe, error)` を実装し、`fixtureRecipes` を複製して返す
      (呼出側で書き換えられても内部 fixture が汚れないよう、スライスの複製を返す)
      (完了判定: 下記 2.4 のテストが pass する)
- [x] 2.4 `recipeservice_test.go` に BDD 風テストを追加する
    - should return all fixture recipes when List is called
    - should return recipes whose Name field is non-empty for every entry
    - should include at least one recipe with two or more tags
    - should include at least one recipe with zero tags
    - should return a fresh slice so mutating the result does not affect subsequent calls
    - (完了判定: `go test ./internal/... ./` が pass する)
- [x] 2.5 `main.go` を更新し、`application.Options.Services` に `RecipeService` を追加登録する
      (完了判定: `task dev` で起動し、devtools console から `RecipeService.List()` が配列を返す。目視は最終 8.6 で確認する)

## 3. Wails バインディング再生成

- [x] 3.1 `wails3 generate bindings` を実行し、`frontend/bindings/github.com/yito0000/meal-cycle/` に
      `recipeservice.js` と `internal/domain/recipe/models.js` (`Recipe`) が生成されることを確認する
      (完了判定: `frontend/bindings/` に該当ファイルが存在する)

## 4. shadcn/ui Table プリミティブ導入

- [x] 4.1 shadcn の docs (https://ui.shadcn.com/docs/components/table) から `Table` / `TableHeader` / `TableBody` /
      `TableRow` / `TableHead` / `TableCell` / `TableCaption` を含む TSX を `frontend/src/components/ui/table.tsx`
      に手動コピーで新設する (CLI は使わない)
      (完了判定: `frontend/src/components/ui/table.tsx` が存在し、`cd frontend && npm run type-check` が通る)

## 5. フロントエンド共通ユーティリティ

- [x] 5.1 `frontend/src/components/recipe/recipe-sort.ts` を新設し、`RecipeSortKey` 型 (`"updated-desc" | "name-asc"`) と
      `compareRecipes(a, b, key)` 関数を定義する
      (完了判定: `cd frontend && npm run type-check` が通る)

## 6. フロントエンド (S02 レシピ一覧)

- [x] 6.1 `frontend/src/components/recipe/recipe-table.tsx` を新設し、shadcn Table を使って
      「名前 / タグ / 更新日」の 3 列を描画する (材料数列は 9.1 で削除、design.md D8 参照)
      (行は props で受け取った Recipe 配列。タグ列は `tags.join(" / ")`、空配列時は空セル)
      (完了判定: 目視で 3 列テーブルが描画される — 最終 8.6 で確認)
- [x] 6.2 `frontend/src/components/recipe/recipe-search-input.tsx` を新設し、制御コンポーネントとして
      検索テキストの `value` / `onChange` を props で受け取る (i18n key `pages.recipes.search.placeholder`)
      (完了判定: 目視で入力欄が表示され、キー入力に反応する — 最終 8.6 で確認)
- [x] 6.3 `frontend/src/components/recipe/recipe-sort-select.tsx` を新設し、
      「更新日 降順」/「名前 昇順」の 2 択セレクト UI を作る (i18n key `pages.recipes.sort.*`)
      (完了判定: 目視で選択切替に反応する — 最終 8.6 で確認)
- [x] 6.4 `frontend/src/routes/recipes.tsx` を更新し、placeholder を実 UI に置換える
      (TanStack Query で `RecipeService.List()` をラップ、`useState` で検索文字列 / ソートキーを管理、
      `useMemo` でフィルタ + ソートを合成して `recipe-table` に渡す)
      (完了判定: 目視で S02 が表示される — 最終 8.6 で確認)
- [x] 6.5 `frontend/src/routes/recipes.tsx` の画面右上に「新しいレシピ」ボタンを配置する
      (`onClick` は no-op、i18n key `pages.recipes.new_button`。ボタンプリミティブは既存 `frontend/src/components/ui/button.tsx`
      があれば使い、無ければ shadcn docs からコピー)
      (完了判定: 目視でボタンが表示され、押しても画面遷移しない — 最終 8.6 で確認)
- [x] 6.6 検索結果が 0 件のときの空状態メッセージを実装する (i18n key `pages.recipes.empty.no_match`)
      (完了判定: 目視で検索欄に該当しない文字列を入れると空状態メッセージが出る — 最終 8.6 で確認)
- [x] 6.7 レシピ 0 件時 (fixture が空の想定は無いが将来対応) の空状態メッセージを実装する
      (i18n key `pages.recipes.empty.no_recipes`)
      (完了判定: 実行時に fixture を空配列に一時差し替えて目視、または該当分岐が JSX に含まれることを目視 — 最終 8.6 で確認)

## 7. i18n

- [x] 7.1 `frontend/src/locales/ja.json` の `pages.recipes.*` を拡張する
      (`title` (既存) / `search.placeholder` / `sort.label` / `sort.updated_desc` / `sort.name_asc` /
      `new_button` / `column.name` / `column.tags` / `column.updated_at` /
      `empty.no_match` / `empty.no_recipes` / `loading` / `load_error` を追加。既存 `placeholder` キーは削除。
      `column.ingredient_count` は 6.1 時点で追加したが 9.2 で削除)
      (完了判定: 目視で日本語が表示される — 最終 8.6 で確認)
- [x] 7.2 `frontend/src/locales/en.json` に同じキーの英訳を追加する
      (完了判定: `ja.json` と key parity を目視確認する)
- [x] 7.3 追加コンポーネントに日本語文字列がハードコードされていないことを grep で確認する
      (完了判定: `grep -rn "[ぁ-んァ-ヶ一-龠]" frontend/src/components/recipe frontend/src/routes/recipes.tsx` が
      no-match で終了する)

## 8. 検証 (test / lint / 型 / build)

- [x] 8.1 Go テストが全て pass することを確認する
      (完了判定: `go test ./internal/... ./` が pass する)
- [x] 8.2 Go 側 lint が通ることを確認する
      (完了判定: `task lint:go` が pass する)
- [x] 8.3 Frontend の TypeScript 型チェックが通ることを確認する
      (完了判定: `cd frontend && npm run type-check` が pass する)
- [x] 8.4 Frontend の Vite ビルド (dev) が通ることを確認する
      (完了判定: `cd frontend && npm run build:dev` が pass する)
- [x] 8.5 Frontend lint (Biome) が通ることを確認する
      (完了判定: `task lint:frontend` が pass する)
- [x] 8.6 `task dev` でアプリを起動し、S02 レシピ一覧が表示されることを目視確認する
      (完了判定: 目視で 3 列テーブルと固定レシピが描画される、検索欄で絞り込みが効く、ソート切替で並びが
      変わる、「新しいレシピ」ボタンが押下無反応で表示される。Wails GUI は環境依存のため apply 内では
      実行せず、ユーザー確認に委ねる)

## 9. 目視レビュー後の UI 調整 (design.md D7 / D8)

- [x] 9.1 「材料数」列をテーブルから削除する
      (対象: `frontend/src/components/recipe/recipe-table.tsx` の TableHead / TableCell 一連)
      (完了判定: 目視で 3 列 (名前 / タグ / 更新日) になる)
- [x] 9.2 i18n から `pages.recipes.column.ingredient_count` を削除する
      (対象: `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`)
      (完了判定: grep で `ingredient_count` が locales に残っていない)
- [x] 9.3 npm に `@radix-ui/react-select` を追加する
      (完了判定: `frontend/package.json` の dependencies に載る)
- [x] 9.4 shadcn の docs (https://ui.shadcn.com/docs/components/select) から Select プリミティブ一式を
      `frontend/src/components/ui/select.tsx` に手動コピーで新設する
      (完了判定: `frontend/src/components/ui/select.tsx` が存在し、型チェックが通る)
- [x] 9.5 `frontend/src/components/recipe/recipe-sort-select.tsx` を shadcn Select ベースに書き換える
      (素の `<select>` を廃止、i18n key は既存を流用)
      (完了判定: 目視でプルダウンを開いた時のオプション文字も Wails Zoom に追従する)
- [x] 9.6 テスト・lint・型・build を再確認する
      (完了判定: `go test ./internal/... ./` / `task lint` / `npm run type-check` /
      `npm run build:dev` が全て pass する)
- [x] 9.7 `task dev` で再度目視確認する (材料数列が消えている、ソートプルダウンのオプションが大きい)
      (完了判定: ユーザー目視 OK)
