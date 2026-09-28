## Why

今はレシピを消す手段が無い。
試しに登録したレシピや、作らなくなったレシピが S02 一覧と献立のレシピ選択ダイアログに残り続け、選ぶたびに目で読み飛ばすことになる。
S03 詳細ドロワーは `add-recipe-details` で入ったが、「削除」ボタンはこの proposal に回してある。

素直に行を消すと、別の問題が起きる。
`meal_plans.recipe_id` には `ON DELETE CASCADE` が付いており、レシピを物理削除すると、そのレシピを使った過去と未来の献立が黙って消える
(docs/reviews/2026-09-25-design-ux-review.md R2)。
消えるのは S01 の表示だけではない。
MVP 後の提案機能「久しぶりに」は献立の履歴から計算するので、その根拠データも一緒に失われる。

本 proposal では、レシピの削除を論理削除として実装する。
削除したレシピは一覧と選択肢から消えるが、行は DB に残り、すでに組んだ献立はそのまま表示される。

## What Changes

- `recipes` テーブルに削除日時の列を足す。値が入っているレシピを「削除済み」とする。
- S03 詳細ドロワーのフッタ (「編集」の隣) に「削除」ボタンを置く。
  - 押すと確認ダイアログを出す。ダイアログは、一覧と選択肢から消えること、献立に入っている分は残ることを伝える
  - 確認すると削除し、ドロワーを閉じ、削除した旨をトーストで知らせる。ドロワーを開く前の画面 (S02 または S01) に留まる
  - 削除に失敗したら、ダイアログを開いたまま失敗した旨を出す
- 削除済みレシピを次の場所から除外する。
  - S02 レシピ一覧 (検索とソートの対象からも外れる)
  - S01 のレシピ選択ダイアログ (割当と入替の候補)
- 削除済みレシピを新たに献立へ割り当てる操作 (割当、入替) をサーバ側でも拒否する。
- 削除済みレシピを使った献立はそのまま残す。
  - S01 のセルには引き続きレシピ名を出す
  - S07 買い物リストは、その献立の材料を従来どおり計上する
  - 既存の献立行に対する移動、コピー、作り置きの操作は従来どおり行える
- S01 の品行メニューから削除済みレシピの S03 を開いた場合は、中身を表示したうえで「削除済み」の表示を出し、「編集」と「削除」のボタンは出さない。
- RecipeService に `Delete(id)` を公開する。削除済みレシピの更新は拒否する。

### Non-goals

- **削除の取り消し (復元)**：確認ダイアログを挟むので、トーストの「元に戻す」やゴミ箱画面は付けない。
  論理削除なので、将来の復元は列を空に戻すだけで実装できる。
- **物理削除 (完全消去)**：DB から行を消す操作は提供しない。`ON DELETE CASCADE` の制約も今回は変えない。
- **S01 のセルでの削除済み表示**：セルのレシピ名に印を付けることはしない。削除済みかどうかは S03 を開けば分かる。
- **確認ダイアログでの使用件数の表示**：「N 件の献立で使われています」は出さない。論理削除では献立が消えないので、件数が判断材料にならない。
- **冷蔵庫の食材名候補からの除外**：S06 の食材名候補は、削除済みレシピの材料名も従来どおり含める (design.md D6)。
- **複数レシピの一括削除**：S02 に選択チェックボックスは付けない。
- **Frontend テスト runner の導入**：Vitest は未導入のまま。フロントは type-check / build:dev / E2E で検証する。

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `recipe`：S03 詳細ドロワーの表示要件を変え、「削除」ボタンと削除済みレシピの表示を加える。
  レシピの削除 (確認、除外範囲、献立の保持、割当の拒否) の要件を追加する。

mealplan の「レシピ選択ダイアログ」要件と shopping の各要件は文面を変えない。
選択肢からの除外と、削除済みレシピの献立を買い物リストが計上し続けることは、recipe 側の削除要件にシナリオとして書く。

## Impact

### 新設ファイル

- `internal/storage/sqlite/migrations/00007_recipes_archived_at.sql`：`recipes.archived_at TEXT` (NULL 許容) を追加する
- `frontend/src/components/recipe/delete-recipe-dialog.tsx`：削除の確認ダイアログ
- `frontend/e2e/recipe-delete.spec.ts`：削除の E2E (対象シナリオは tasks.md で列挙)

### 更新ファイル

- `internal/domain/recipe/recipe.go`：`Recipe` に `ArchivedAt string` (`json:"archivedAt"`、空文字 = 未削除) を追加
- `internal/storage/sqlite/recipe_repo.go`：`Archive(ctx, id)` を追加。`List` は削除済みを除外、`GetByID` は削除済みも返し `ArchivedAt` を埋める。
  `Update` は削除済みなら `ErrRecipeArchived` を返す
- `internal/storage/sqlite/recipe_repo_test.go`：`Archive` と `List` / `GetByID` / `Update` の削除済みまわりのテストを追加
- `internal/storage/sqlite/mealplan_repo.go`：`Assign` と `Replace` で削除済みレシピを `mealplan.ErrUnknownRecipe` として拒否する
- `internal/storage/sqlite/mealplan_repo_test.go`：`Assign` / `Replace` の削除済み拒否と、削除済みレシピの献立が `ListPeriod` に残るテストを追加
- `internal/storage/sqlite/shopping_repo_test.go`：削除済みレシピの献立の材料が計上されるテストを追加
- `internal/service/recipe.go`：`Delete(ctx, id)` を追加
- `internal/service/recipe_test.go`：`TestRecipeService_Delete` を追加
- `cmd/e2e-server/dispatch_test.go`：`RecipeService/Delete` の転送テストを追加
- `frontend/e2e/bindings-shim/recipeservice.ts`：`Delete` を追加
- `frontend/src/components/recipe/recipe-detail-drawer.tsx`：「削除」ボタン、確認ダイアログの開閉、削除済み表示、削除後のクエリ無効化とトースト
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：`pages.recipes.detail.delete` / `pages.recipes.detail.archived` /
  `pages.recipes.delete_confirm.*` を追加
- `frontend/e2e/helpers.ts`：`deleteRecipeFromDetail` (ドロワーから削除して確認まで行う) を追加
- `frontend/e2e/recipe-detail.spec.ts`：「削除」ボタンを出さないことを確かめる既存の検証を、「削除」ボタンを出す検証に差し替える
- `docs/product.md`：§4 S03 の削除の記述を、論理削除と「開く前の画面に留まる」挙動に合わせて直す
- `docs/design.md`：§4 データモデルに `recipes.archived_at` を追記する
- `docs/reviews/2026-09-25-design-ux-review.md`：R2 を対応済みにする

### 生成物 (再生成のみ)

- `frontend/bindings/`：`RecipeService.Delete` と `Recipe.archivedAt` の追加に伴い `wails3 generate bindings -clean=true -ts -i` で再生成

### migration

- `00007_recipes_archived_at.sql`：`ALTER TABLE recipes ADD COLUMN archived_at TEXT`。Down は列を落とす

### 後続 proposal への引き継ぎ

- 復元 UI を足すなら、`RecipeRepo` に `archived_at = NULL` へ戻すメソッドを足し、削除済みレシピの一覧画面を用意する
- 提案機能 (「久しぶりに」など) は献立履歴を読むとき、削除済みレシピを候補に出すかを決める必要がある。
  履歴の集計には含め、提案の候補からは外すのが自然と考えている
- S01 のセルで削除済みを示したくなったら、`mealplan.Entry` に削除済みフラグを足し、`ListPeriod` の JOIN で埋める
