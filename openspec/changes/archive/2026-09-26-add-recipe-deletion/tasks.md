## 1. スキーマとドメイン

- [x] 1.1 `internal/storage/sqlite/migrations/00007_recipes_archived_at.sql` を新設する (design.md D2)
    - Up: `ALTER TABLE recipes ADD COLUMN archived_at TEXT;`
    - Down: `ALTER TABLE recipes DROP COLUMN archived_at;`
    - 完了判定: `go test ./internal/storage/sqlite/ -run 'TestOpenAndMigrate|TestMigrateIsIdempotent'` が通る
- [x] 1.2 `internal/domain/recipe/recipe.go` を更新し、`Recipe` に `ArchivedAt string` (`json:"archivedAt"`) を追加する。
      空文字が未削除を表す旨を struct のコメントに 1 行足す。完了判定: `go build .` が通る

## 2. Repository

- [x] 2.1 `internal/storage/sqlite/recipe_repo.go` に `ErrRecipeArchived = errors.New("recipe archived")` を追加する。
      完了判定: `go build .` が通る
- [x] 2.2 `internal/storage/sqlite/recipe_repo.go` の `List` に `WHERE r.archived_at IS NULL` を足す。
      除外が要るのは献立を組む入口だけである旨を 1 行コメントで残す (design.md D4)。完了判定: 2.7 のテストが通る
- [x] 2.3 `internal/storage/sqlite/recipe_repo.go` の `GetByID` で `archived_at` を読み、NULL なら空文字として `ArchivedAt` に入れる。
      削除済みも返す。完了判定: 2.7 のテストが通る
- [x] 2.4 `internal/storage/sqlite/recipe_repo.go` の `Update` で、既存行が削除済みなら `ErrRecipeArchived` を返して行を変えない。
      完了判定: 2.7 のテストが通る
- [x] 2.5 `internal/storage/sqlite/recipe_repo.go` に `Archive(ctx, id string) error` を追加する (design.md D3)
    - 該当 ID が無ければ `ErrRecipeNotFound`
    - 未削除なら `archived_at` に `time.Now().UTC().Format(time.RFC3339)` を入れる。`updated_at` は変えない
    - 削除済みなら何もせず nil を返す
    - 完了判定: 2.7 のテストが通る
- [x] 2.6 `internal/storage/sqlite/mealplan_repo.go` の `Assign` と `Replace` で、書き込み前に
      `SELECT 1 FROM recipes WHERE id = ? AND archived_at IS NULL` を確かめ、該当しなければ `mealplan.ErrUnknownRecipe` を返す (design.md D4)。
      `Replace` は確認と UPDATE を同じトランザクションに入れる。完了判定: 2.8 のテストが通る
- [x] 2.7 `internal/storage/sqlite/recipe_repo_test.go` にテストを追加する
    - `TestRecipeRepo_Archive`：未削除のレシピを削除すると `GetByID` の `ArchivedAt` が RFC3339 で埋まる
    - `TestRecipeRepo_Archive`：削除しても `UpdatedAt` は変わらない
    - `TestRecipeRepo_Archive`：削除済みのレシピをもう一度削除しても nil を返し、`ArchivedAt` は最初の値のまま
    - `TestRecipeRepo_Archive`：存在しない ID は `ErrRecipeNotFound`
    - `TestRecipeRepo_Archive`：削除しても `recipe_ingredients` の行は残り、`GetByID` の `Ingredients` が返る
    - `TestRecipeRepo_List`：削除済みのレシピは返さず、未削除のレシピだけを返す
    - `TestRecipeRepo_List`：全レシピが削除済みなら空スライスを返す
    - `TestRecipeRepo_GetByID`：未削除のレシピは `ArchivedAt` が空文字
    - `TestRecipeRepo_Update`：削除済みのレシピは `ErrRecipeArchived` を返し、名前と `UpdatedAt` が変わらない
    - `TestRecipeRepo_Archive`：DB を閉じて開き直しても削除済みのまま (再起動後も一覧に出ない)
    - 完了判定: `go test ./internal/storage/sqlite/ -run 'TestRecipeRepo'` が通る
- [x] 2.8 `internal/storage/sqlite/mealplan_repo_test.go` にテストを追加する
    - `TestMealPlanRepo_Assign`：削除済みレシピの割当は `mealplan.ErrUnknownRecipe` を返し、行が増えない
    - `TestMealPlanRepo_Replace`：削除済みレシピへの入替は `mealplan.ErrUnknownRecipe` を返し、元の `recipe_id` のまま
    - `TestMealPlanRepo_ListPeriod`：割当後にレシピを削除しても、その行がレシピ名つきで返る
    - `TestMealPlanRepo_CopyLeftover`：削除済みレシピの行も作り置きとして広げられる
    - 完了判定: `go test ./internal/storage/sqlite/ -run 'TestMealPlanRepo'` が通る
- [x] 2.9 `internal/storage/sqlite/shopping_repo_test.go` にテストを追加する
    - `TestShoppingRepo_ListPlannedBatches`：削除済みレシピを使った献立の材料が計上され、献立一覧にそのレシピ名が含まれる
    - 完了判定: `go test ./internal/storage/sqlite/ -run 'TestShoppingRepo'` が通る

## 3. Service と bindings

- [x] 3.1 `internal/service/recipe.go` に `Delete(ctx, id string) error` を追加し、`s.repo.Archive` に委譲する (design.md D3)。
      `RecipeService` の struct コメントの公開メソッド一覧に `Delete` を足す。完了判定: `go build .` が通る
- [x] 3.2 `internal/service/recipe_test.go` に `TestRecipeService_Delete` を追加する
    - should hide the recipe from List when deleted
    - should keep returning the recipe from GetByID with archivedAt when deleted
    - should return ErrRecipeNotFound when id does not exist
    - should succeed without error when the recipe is already deleted
    - 完了判定: `go test ./internal/service/ -run TestRecipeService_Delete` が通る
- [x] 3.3 `internal/service/recipe_test.go` の `TestRecipeService_Update` にケースを追加する
    - should return ErrRecipeArchived when the recipe is deleted
    - 完了判定: `go test ./internal/service/ -run TestRecipeService_Update` が通る
- [x] 3.4 `cmd/e2e-server/dispatch_test.go` に `RecipeService/Delete` の転送テストを追加する
    - 作ったレシピを `/api/RecipeService/Delete` で消すと、`/api/RecipeService/List` に出なくなる
    - 完了判定: `go test ./cmd/...` が通る
- [x] 3.5 `wails3 generate bindings -clean=true -ts -i` で `frontend/bindings/` を再生成する。
      完了判定: 生成された `RecipeService` に `Delete` があり、`Recipe` 型に `archivedAt` がある
- [x] 3.6 `frontend/e2e/bindings-shim/recipeservice.ts` に `Delete(id: string)` を追加する (戻り値は `call<void>`)。
      完了判定: `cd frontend && npm run type-check` が通る

## 4. Frontend

- [x] 4.1 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に design.md D8 の key を追加する
    - `pages.recipes.detail.delete` / `pages.recipes.detail.archived`
    - `pages.recipes.delete_confirm.title` / `body` / `cancel` / `confirm` / `error` / `done`
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.2 `frontend/src/components/recipe/delete-recipe-dialog.tsx` を新設する (design.md D7)
    - props: `open` / `recipeName` / `pending` / `error` / `onConfirm` / `onCancel`
    - `unsaved-changes-dialog.tsx` と同じ Radix Dialog の構成にし、確認ボタンは `variant="destructive"`
    - `pending` の間は両ボタンを `disabled` にする
    - `error` が真なら本文の下に `pages.recipes.delete_confirm.error` を出す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.3 `frontend/src/components/recipe/recipe-detail-drawer.tsx` を更新し、フッタの「編集」の隣に「削除」ボタンを置く。
      押すと `DeleteRecipeDialog` を開く。完了判定: `task dev` で「削除」を押すと確認ダイアログが開くことを目視
- [x] 4.4 `frontend/src/components/recipe/recipe-detail-drawer.tsx` に `RecipeService.Delete` を呼ぶ `useMutation` を追加する (design.md D7)
    - 成功時: `["recipes", "list"]` を無効化する
    - 成功時: `["recipes", "detail", id]` を `refetchType: "none"` で無効化する (再取得はしない)
    - 成功時: `toast` で `pages.recipes.delete_confirm.done` を出す
    - 成功時: 確認ダイアログを閉じ、`onClose()` を呼ぶ
    - 失敗時: 確認ダイアログを開いたまま `error` を立てる
    - 完了判定: `task dev` で削除するとドロワーが閉じ、一覧から行が消え、トーストが出ることを目視
- [x] 4.5 `frontend/src/components/recipe/recipe-detail-drawer.tsx` で、`recipe.archivedAt` が空でないときの表示を実装する (design.md D5)
    - `SheetDescription` に `pages.recipes.detail.archived` を出す
    - `SheetFooter` を描画しない (「編集」「削除」を出さない)
    - 完了判定: `cd frontend && npm run type-check` が通る

## 5. E2E

- [x] 5.1 `frontend/e2e/helpers.ts` に `deleteRecipeFromDetail(page: Page)` を追加する。
      開いているドロワーで「削除」を押し、確認ダイアログの「削除」を押し、ドロワーが閉じるまで待つ。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.2 `frontend/e2e/helpers.ts` に `deleteRecipe(page: Page, id: string)` を追加する。
      `/api/RecipeService/Delete` を直接呼ぶ (S01 側の前提づくり用)。完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.3 `frontend/e2e/recipe-detail.spec.ts` の「詳細ドロワーに基本情報と材料と手順とメモが表示される」を更新し、
      「削除」ボタンが 0 件であることの検証を、`d.delete` のボタンが見えることの検証に差し替える。
      完了判定: `task test:e2e` でこのテストが通る
- [x] 5.4 `frontend/e2e/recipe-delete.spec.ts` を新設し、次の Scenario を E2E で確かめる (describe 名は Requirement 名と一字一句揃える)
    - describe「レシピの削除」
      - 確認してレシピを削除すると一覧から消える
      - 確認ダイアログでキャンセルすると削除しない
      - S01 から開いたドロワーで削除すると S01 に留まる
    - describe「削除済みレシピの除外範囲」
      - 削除済みレシピはレシピ選択ダイアログに出ない
      - 全レシピを削除すると空状態になる
    - describe「削除済みレシピを使った献立の保持」
      - 削除したレシピの品が S01 に残る
    - describe「S03 レシピ詳細ドロワーの表示」
      - 削除済みレシピは内容と削除済みの旨を表示し、操作ボタンを出さない
    - 完了判定: `task test:e2e` でこのファイルの全テストが通る
- [x] 5.5 次の Scenario は E2E に含めず、Go のテストで確かめる。完了判定: 対応するテストが 2.7 / 2.8 / 2.9 / 3.3 に含まれている
    - 削除したレシピは再起動後も一覧に出ない (2.7)
    - 削除済みレシピを献立に割り当てる要求は拒否される (2.8)
    - 削除済みレシピの編集の保存は拒否される (3.3)
    - 削除したレシピの材料が買い物リストに計上される (2.9)

## 6. ドキュメント

- [x] 6.1 `docs/product.md` §4 S03 の `削除` の行を書き換える。
      確認ダイアログ → 論理削除 → ドロワーを閉じて開く前の画面 (S02 / S01) に留まる、献立には残る、の順で書き、
      「`add-recipe-deletion` で追加する」の注記を消す。完了判定: 目視
- [x] 6.2 `docs/design.md` §4 のデータモデルに `recipes.archived_at` (NULL = 未削除) と、論理削除を選んだ理由への参照
      (本 change の design.md D1) を追記する。完了判定: 目視
- [x] 6.3 `docs/reviews/2026-09-25-design-ux-review.md` の R2 のチェックボックスを `[x]` にし、論理削除を採った旨を 1 行足す。完了判定: 目視
- [x] 6.4 `openspec/specs/recipe/spec.md` の Purpose から「削除は後続の proposal で追加する」を消し、削除を含む文に直す。完了判定: 目視

## 7. 検証

- [x] 7.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass
- [x] 7.2 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0
- [x] 7.3 `cd frontend && npm run build:dev` を実行する。完了判定: ビルド成功
- [x] 7.4 `task test:e2e` を実行する。完了判定: `check:scenarios` を含む全テストが pass
- [x] 7.5 `task lint` を実行する。完了判定: Go と Frontend の両方でエラー 0
- [x] 7.6 `openspec validate add-recipe-deletion` を実行する。完了判定: valid
