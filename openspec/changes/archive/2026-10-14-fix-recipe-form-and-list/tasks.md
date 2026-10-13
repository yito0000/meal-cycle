## 1. 未保存の確認

- [x] 1.1 `frontend/src/components/recipe/recipe-edit-form.tsx` (更新) に `useBlocker` を足す (design.md D1)
    - `shouldBlockFn: () => form.isDirty && !leavingAfterSave.current`、`withResolver: true`、`enableBeforeUnload: false`
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 1.2 `frontend/src/components/recipe/recipe-edit-form.tsx` (更新) の `UnsavedChangesDialog` を、`blocker.status === "blocked"` で開き、
  `onDiscard` で `blocker.proceed()`、`onKeepEditing` で `blocker.reset()` を呼ぶ形に変える。完了判定: `cd frontend && npm run type-check` が通る
- [x] 1.3 `frontend/src/components/recipe/recipe-edit-form.tsx` (更新) から `pendingLeave`、`leave`、`guardedBreadcrumb` を消し、
  キャンセルとパンくずは移動先の関数をそのまま呼ぶ。完了判定: `cd frontend && npm run type-check` が通る
- [x] 1.4 `frontend/src/components/recipe/recipe-edit-form.tsx` (更新) の `saveMutation` に、`onMutate` で `leavingAfterSave.current = true`、
  `onError` で `false` に戻す処理を足す (design.md D2)。`onError` に、戻さないと未保存の確認が出なくなる旨を 1 行のコメントで書く。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 1.5 `frontend/e2e/recipe-edit.spec.ts` (更新) の describe「未保存変更のキャンセル確認」にシナリオを足す
    - dirty 状態でサイドバーを押すと確認される
    - サイドバーからの確認で破棄を選ぶと押した画面へ移る
    - サイドバーからの確認で編集を続けるを選ぶと入力が残る
    - dirty でない状態のサイドバーは確認をスキップする
    - 保存に成功したあとの移動では確認されない
    - 完了判定: `task test:e2e` で上記と、既存の「dirty 状態でキャンセルを押すと確認される」「編集モードで変更を破棄すると S03 に戻る」が pass する

## 2. 数値欄

- [x] 2.1 `frontend/src/components/recipe/use-recipe-form.ts` (更新) に `IngredientFormRow` を足し、`RecipeFormValues` の `servings` と材料行の `amount` を文字列にする
  (design.md D3)。完了判定: 2.2〜2.4 と合わせて `npm run type-check` が通る
- [x] 2.2 `frontend/src/components/recipe/use-recipe-form.ts` (更新) の `blankFormValues` と `recipeToFormValues` を直す
    - `servings` は `"1"` / `String(r.servings)`
    - `amount` は、0 なら空文字、それ以外は `String(amount)`
    - 完了判定: 2.3、2.4 と合わせて `npm run type-check` が通る
- [x] 2.3 `frontend/src/components/recipe/recipe-ingredient-row.tsx` (更新) の props を `IngredientFormRow` にし、量の欄の `value` と `onChange` を文字列のまま受け渡す。
  完了判定: 2.4 と合わせて `npm run type-check` が通る
- [x] 2.4 `frontend/src/components/recipe/recipe-edit-form.tsx` (更新) を直す
    - 人数欄の `onChange` は `e.target.value` をそのまま入れる
    - 材料行を足すときの初期値を `{ name: "", amount: "", unit: "" }` にする
    - `handleSave` で、`servings` を `Number` に、`amount` を空なら 0、それ以外は `Number` に直してから `validate` と `onSave` に渡す
    - `servings` が空、または整数でないときは、既存の人数のエラー (`servings_out_of_range`) にする
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.5 `frontend/e2e/recipe-edit.spec.ts` (更新) に describe「数値欄の入力途中の保持」を足す
    - 人数を消して打ち直せる (`fill` ではなく、Backspace と `pressSequentially` で 1 文字ずつ操作する)
    - 人数が空のままでは保存できない
    - 量を消して打ち直せる
    - 量が空の材料は量 0 として保存される
    - 完了判定: `task test:e2e` で上記が pass する

## 3. S02 の一覧

- [x] 3.1 `frontend/src/lib/format-date.ts` (更新) に `formatTimestampDate(rfc3339: string): string` を足す (design.md D4)。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.2 `frontend/src/components/recipe/recipe-detail.tsx` (更新) の更新日の整形を `formatTimestampDate` に置き換え、使わなくなった import を消す。
  完了判定: `task test:e2e` の describe「S03 レシピ詳細画面の表示」が pass する
- [x] 3.3 `frontend/src/components/recipe/recipe-table.tsx` (更新) の更新日のセルを `formatTimestampDate(r.updatedAt)` にする。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.4 `frontend/src/components/recipe/recipe-table.tsx` (更新) のレシピ名を `<button type="button">` で包む (design.md D5)
    - `onClick` で `stopPropagation` してから `onRowClick` を呼ぶ
    - 見た目は文字のままにし、`focus-visible` のときだけ輪郭を出す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.5 `frontend/e2e/recipe-listing.spec.ts` (更新) の describe「レシピ一覧の初期表示」に「更新日は年月日だけで表示される」を足す。完了判定: `task test:e2e` で上記が pass する
- [x] 3.6 `frontend/e2e/recipe-listing.spec.ts` (更新) に describe「S02 の行のキーボード操作」を足す
    - レシピ名にフォーカスして Enter で詳細を開く
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.7 `frontend/e2e/helpers.ts` (更新) の `openRecipeDetail` など、S02 の行をレシピ名の文字で探している locator が、ボタン化のあとも 1 つに決まることを確かめる。
  完了判定: `task test:e2e` の既存のレシピ系のテストが pass する

## 4. ドキュメントと検証

- [x] 4.1 `docs/product.md` (更新) §4 S02 の「テーブル形式（名前 / タグ / 材料数 / 更新日）」を「名前 / タグ / 更新日」に直す。完了判定: 差分を目視で確認する
- [x] 4.2 `docs/product.md` (更新) §4 S04 の未保存の確認に、サイドバーで別の画面へ移るときも同じ確認を出すことを足す。完了判定: 差分を目視で確認する
- [x] 4.3 `docs/reviews/2026-10-14-design-ux-review.md` (更新) の R28、R32、R33 にチェックを入れ、`fix-recipe-form-and-list` で対応した旨を追記する。完了判定: 差分を目視で確認する
- [x] 4.4 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 4.5 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 4.6 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 4.7 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 4.8 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 4.9 `task build:win` で作った `.exe` で、S04 の手順欄に IME で日本語を入力してからサイドバーを押す。
  完了判定: 確認ダイアログが開き、「編集を続ける」で入力が残ることを目視で確認する
- [x] 4.10 `openspec validate fix-recipe-form-and-list --strict` を実行する。完了判定: エラー 0 件
